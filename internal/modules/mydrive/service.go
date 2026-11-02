package mydrive

import (
	"bufio"
	"cloud-drive/internal/platform/storage"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

var ErrInvalidInput = errors.New("input tidak valid")

type Service struct {
	repo      *Repository
	storage   storage.Provider
	maxUpload int64
}

func NewService(r *Repository, s storage.Provider, max int64) *Service {
	return &Service{repo: r, storage: s, maxUpload: max}
}
func (s *Service) Browse(ctx context.Context, userID uuid.UUID, parentID *uuid.UUID, q, sort string) (Page, error) {
	d, err := s.repo.PersonalDrive(ctx, userID)
	if err != nil {
		return Page{}, err
	}
	page := Page{Drive: d}
	if parentID != nil {
		n, err := s.repo.Node(ctx, d.ID, *parentID)
		if err != nil || n.Type != "folder" {
			return Page{}, ErrNotFound
		}
		page.Parent = &n
		crumbs := []Crumb{}
		current := n
		for {
			id := current.ID
			crumbs = append([]Crumb{{ID: &id, Name: current.Name}}, crumbs...)
			if current.ParentID == nil {
				break
			}
			current, err = s.repo.Node(ctx, d.ID, *current.ParentID)
			if err != nil {
				return Page{}, err
			}
		}
		page.Breadcrumbs = crumbs
	}
	page.Nodes, err = s.repo.List(ctx, d.ID, parentID, strings.TrimSpace(q), sort)
	return page, err
}
func (s *Service) Usage(ctx context.Context, userID uuid.UUID) (int64, error) {
	drive, err := s.repo.PersonalDrive(ctx, userID)
	if err != nil {
		return 0, err
	}
	return s.repo.Usage(ctx, drive.ID)
}
func (s *Service) CreateFolder(ctx context.Context, userID uuid.UUID, parentID *uuid.UUID, name string) error {
	name = safeName(name)
	if name == "" {
		return ErrInvalidInput
	}
	d, err := s.repo.PersonalDrive(ctx, userID)
	if err != nil {
		return err
	}
	if parentID != nil {
		n, e := s.repo.Node(ctx, d.ID, *parentID)
		if e != nil || n.Type != "folder" {
			return ErrNotFound
		}
	}
	return s.repo.CreateFolder(ctx, d.ID, userID, parentID, name)
}
func (s *Service) Upload(ctx context.Context, userID uuid.UUID, parentID *uuid.UUID, filename string, src io.Reader) error {
	filename = safeName(filename)
	if filename == "" {
		return ErrInvalidInput
	}
	d, err := s.repo.PersonalDrive(ctx, userID)
	if err != nil {
		return err
	}
	if parentID != nil {
		n, e := s.repo.Node(ctx, d.ID, *parentID)
		if e != nil || n.Type != "folder" {
			return ErrNotFound
		}
	}
	reader := bufio.NewReader(src)
	header, _ := reader.Peek(512)
	mime := http.DetectContentType(header)
	if blocked(filename, mime) {
		return fmt.Errorf("%w: tipe file tidak diizinkan", ErrInvalidInput)
	}
	key := strings.ReplaceAll(uuid.NewString(), "-", "")
	limited := io.LimitReader(reader, s.maxUpload+1)
	size, checksum, err := s.storage.Put(ctx, key, limited)
	if err != nil {
		return err
	}
	if size > s.maxUpload {
		_ = s.storage.Delete(ctx, key)
		return fmt.Errorf("%w: ukuran file melebihi batas", ErrInvalidInput)
	}
	if err = s.repo.CreateFile(ctx, d.ID, userID, parentID, filename, mime, key, checksum, size); err != nil {
		_ = s.storage.Delete(ctx, key)
		return err
	}
	return nil
}
func (s *Service) Download(ctx context.Context, userID, nodeID uuid.UUID) (Download, io.ReadSeekCloser, error) {
	d, err := s.repo.PersonalDrive(ctx, userID)
	if err != nil {
		return Download{}, nil, err
	}
	item, err := s.repo.Download(ctx, d.ID, nodeID)
	if err != nil {
		return Download{}, nil, err
	}
	file, err := s.storage.Open(ctx, item.StorageKey)
	return item, file, err
}
func (s *Service) Rename(ctx context.Context, userID, nodeID uuid.UUID, name string) error {
	name = safeName(name)
	if name == "" {
		return ErrInvalidInput
	}
	d, err := s.repo.PersonalDrive(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.Rename(ctx, d.ID, userID, nodeID, name)
}
func (s *Service) Trash(ctx context.Context, userID, nodeID uuid.UUID) error {
	d, err := s.repo.PersonalDrive(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.Trash(ctx, d.ID, userID, nodeID)
}
func (s *Service) CreateShare(ctx context.Context, userID, nodeID uuid.UUID, expiresInDays int, allowDownload bool) (ShareLink, error) {
	drive, err := s.repo.PersonalDrive(ctx, userID)
	if err != nil {
		return ShareLink{}, err
	}
	if _, err = s.repo.Node(ctx, drive.ID, nodeID); err != nil {
		return ShareLink{}, err
	}
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return ShareLink{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	var expiresAt *time.Time
	if expiresInDays > 0 {
		if expiresInDays > 365 {
			return ShareLink{}, ErrInvalidInput
		}
		value := time.Now().UTC().Add(time.Duration(expiresInDays) * 24 * time.Hour)
		expiresAt = &value
	}
	if err = s.repo.CreateShareLink(ctx, nodeID, userID, hash[:], expiresAt, allowDownload); err != nil {
		return ShareLink{}, err
	}
	return ShareLink{Token: token, ExpiresAt: expiresAt, AllowDownload: allowDownload}, nil
}
func (s *Service) RevokeShare(ctx context.Context, userID, nodeID uuid.UUID) error {
	drive, err := s.repo.PersonalDrive(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.RevokeShareLink(ctx, drive.ID, nodeID)
}
func (s *Service) ShareStatuses(ctx context.Context, userID uuid.UUID) ([]ShareStatus, error) {
	drive, err := s.repo.PersonalDrive(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.ShareStatuses(ctx, drive.ID)
}
func (s *Service) ResolveShare(ctx context.Context, token string) (SharedRoot, error) {
	if len(token) < 32 || len(token) > 128 {
		return SharedRoot{}, ErrNotFound
	}
	hash := sha256.Sum256([]byte(token))
	return s.repo.SharedRoot(ctx, hash[:])
}
func (s *Service) SharedBrowse(ctx context.Context, root SharedRoot, folderID *uuid.UUID) (Node, []Node, error) {
	if root.Type == "file" {
		return root.Node, nil, nil
	}
	id := root.ID
	if folderID != nil {
		id = *folderID
	}
	folder, err := s.repo.SharedFolder(ctx, root, id)
	if err != nil {
		return Node{}, nil, err
	}
	nodes, err := s.repo.SharedChildren(ctx, root, id)
	return folder, nodes, err
}
func (s *Service) SharedDownload(ctx context.Context, root SharedRoot, nodeID uuid.UUID) (Download, io.ReadSeekCloser, error) {
	file, err := s.repo.SharedFile(ctx, root, nodeID)
	if err != nil {
		return Download{}, nil, err
	}
	src, err := s.storage.Open(ctx, file.StorageKey)
	return file, src, err
}
func safeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\") || len(name) > 255 {
		return ""
	}
	return filepath.Base(name)
}
func blocked(name, mime string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".exe", ".dll", ".bat", ".cmd", ".com", ".ps1", ".sh", ".msi", ".scr", ".js", ".jar":
		return true
	}
	return mime == "application/x-msdownload" || mime == "application/x-dosexec"
}
