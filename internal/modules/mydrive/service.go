package mydrive

import (
	"bufio"
	"cloud-drive/internal/platform/storage"
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"path/filepath"
	"strings"
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
