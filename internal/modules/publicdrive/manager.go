package publicdrive

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"cloud-drive/internal/middleware"
	"cloud-drive/internal/modules/auth"
	"cloud-drive/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var ErrManagerNotFound = errors.New("resource not found")
var ErrManagerInput = errors.New("input tidak valid")

func (r *Repository) ManagedDrive(ctx context.Context, userID uuid.UUID, slug string) (Drive, error) {
	var d Drive
	err := r.db.QueryRowContext(ctx, `SELECT d.id,d.name,o.id,o.slug,o.name FROM drive_drives d JOIN org_organizations o ON o.id=d.organization_id JOIN org_memberships m ON m.organization_id=o.id AND m.user_id=? AND m.status='active' WHERE o.slug=? AND o.active=1 AND d.type='public'`, userID.String(), slug).Scan(&d.ID, &d.Name, &d.Organization.ID, &d.Organization.Slug, &d.Organization.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return d, ErrManagerNotFound
	}
	return d, err
}
func (r *Repository) ManagerNode(ctx context.Context, driveID, nodeID uuid.UUID) (Node, error) {
	var n Node
	var parent sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT id,parent_id,name,type,COALESCE(mime_type,''),size_bytes,updated_at FROM drive_nodes WHERE id=? AND drive_id=? AND deleted_at IS NULL`, nodeID.String(), driveID.String()).Scan(&n.ID, &parent, &n.Name, &n.Type, &n.MIME, &n.Size, &n.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return n, ErrManagerNotFound
	}
	if parent.Valid {
		id, _ := uuid.Parse(parent.String)
		n.ParentID = &id
	}
	return n, err
}
func (r *Repository) CreateManagerFolder(ctx context.Context, driveID, userID uuid.UUID, parent *uuid.UUID, name string) error {
	var p any
	if parent != nil {
		p = parent.String()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO drive_nodes(id,drive_id,parent_id,name,type,status,created_by,updated_by) VALUES(?,?,?,?, 'folder','ready',?,?)`, uuid.NewString(), driveID.String(), p, name, userID.String(), userID.String())
	return err
}
func (r *Repository) CreateManagerFile(ctx context.Context, driveID, userID uuid.UUID, parent *uuid.UUID, name, mimeType, key, checksum string, size int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	objectID, nodeID := uuid.NewString(), uuid.NewString()
	if _, err = tx.ExecContext(ctx, `INSERT INTO storage_objects(id,storage_key,size_bytes,checksum_sha256,state) VALUES(?,?,?,?,'ready')`, objectID, key, size, checksum); err != nil {
		return err
	}
	var p any
	if parent != nil {
		p = parent.String()
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO drive_nodes(id,drive_id,parent_id,name,type,mime_type,size_bytes,storage_object_id,status,created_by,updated_by) VALUES(?,?,?,?, 'file',?,?,?,'ready',?,?)`, nodeID, driveID.String(), p, name, mimeType, size, objectID, userID.String(), userID.String()); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO drive_file_versions(id,node_id,version_no,object_id,size_bytes,checksum_sha256,created_by) VALUES(?,?,1,?,?,?,?)`, uuid.NewString(), nodeID, objectID, size, checksum, userID.String()); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) ManagerFile(ctx context.Context, driveID, nodeID uuid.UUID) (File, error) {
	var f File
	var p sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT n.id,n.parent_id,n.name,n.type,COALESCE(n.mime_type,''),n.size_bytes,n.updated_at,o.storage_key FROM drive_nodes n JOIN storage_objects o ON o.id=n.storage_object_id WHERE n.id=? AND n.drive_id=? AND n.type='file' AND n.deleted_at IS NULL AND n.status='ready' AND o.state='ready'`, nodeID.String(), driveID.String()).Scan(&f.ID, &p, &f.Name, &f.Type, &f.MIME, &f.Size, &f.UpdatedAt, &f.StorageKey)
	if errors.Is(err, sql.ErrNoRows) {
		return f, ErrManagerNotFound
	}
	return f, err
}
func (r *Repository) ManagerRename(ctx context.Context, driveID, userID, nodeID uuid.UUID, name string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE drive_nodes SET name=?,updated_by=? WHERE id=? AND drive_id=? AND deleted_at IS NULL`, name, userID.String(), nodeID.String(), driveID.String())
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrManagerNotFound
	}
	return nil
}
func (r *Repository) ManagerTrash(ctx context.Context, driveID, userID, nodeID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `UPDATE drive_nodes SET deleted_at=UTC_TIMESTAMP(6),updated_by=? WHERE id=? AND drive_id=? AND deleted_at IS NULL`, userID.String(), nodeID.String(), driveID.String())
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrManagerNotFound
	}
	return nil
}

type ManagerService struct {
	repo      *Repository
	storage   storage.Provider
	slug      string
	maxUpload int64
}

func NewManagerService(r *Repository, s storage.Provider, slug string, max int64) *ManagerService {
	return &ManagerService{repo: r, storage: s, slug: slug, maxUpload: max}
}
func (s *ManagerService) Browse(ctx context.Context, userID uuid.UUID, parent *uuid.UUID, q, sort string) (ManagerPage, error) {
	d, err := s.repo.ManagedDrive(ctx, userID, s.slug)
	if err != nil {
		return ManagerPage{}, err
	}
	page := ManagerPage{Drive: d}
	if parent != nil {
		n, e := s.repo.ManagerNode(ctx, d.ID, *parent)
		if e != nil || n.Type != "folder" {
			return page, ErrManagerNotFound
		}
		page.Parent = &n
		current := n
		for {
			id := current.ID
			page.Breadcrumbs = append([]Crumb{{ID: &id, Name: current.Name}}, page.Breadcrumbs...)
			if current.ParentID == nil {
				break
			}
			current, err = s.repo.ManagerNode(ctx, d.ID, *current.ParentID)
			if err != nil {
				return page, err
			}
		}
	}
	page.Nodes, err = s.repo.List(ctx, d.Organization.ID, parent, q, "", sort)
	return page, err
}
func (s *ManagerService) CreateFolder(ctx context.Context, userID uuid.UUID, parent *uuid.UUID, name string) error {
	name = managerSafeName(name)
	if name == "" {
		return ErrManagerInput
	}
	d, err := s.repo.ManagedDrive(ctx, userID, s.slug)
	if err != nil {
		return err
	}
	if parent != nil {
		n, e := s.repo.ManagerNode(ctx, d.ID, *parent)
		if e != nil || n.Type != "folder" {
			return ErrManagerNotFound
		}
	}
	return s.repo.CreateManagerFolder(ctx, d.ID, userID, parent, name)
}
func (s *ManagerService) Upload(ctx context.Context, userID uuid.UUID, parent *uuid.UUID, name string, src io.Reader) error {
	name = managerSafeName(name)
	if name == "" {
		return ErrManagerInput
	}
	d, err := s.repo.ManagedDrive(ctx, userID, s.slug)
	if err != nil {
		return err
	}
	if parent != nil {
		n, e := s.repo.ManagerNode(ctx, d.ID, *parent)
		if e != nil || n.Type != "folder" {
			return ErrManagerNotFound
		}
	}
	reader := bufio.NewReader(src)
	header, _ := reader.Peek(512)
	mimeType := http.DetectContentType(header)
	if managerBlocked(name, mimeType) {
		return ErrManagerInput
	}
	key := strings.ReplaceAll(uuid.NewString(), "-", "")
	size, checksum, err := s.storage.Put(ctx, key, io.LimitReader(reader, s.maxUpload+1))
	if err != nil {
		return err
	}
	if size > s.maxUpload {
		_ = s.storage.Delete(ctx, key)
		return ErrManagerInput
	}
	if err = s.repo.CreateManagerFile(ctx, d.ID, userID, parent, name, mimeType, key, checksum, size); err != nil {
		_ = s.storage.Delete(ctx, key)
	}
	return err
}
func (s *ManagerService) File(ctx context.Context, userID, nodeID uuid.UUID) (File, io.ReadSeekCloser, error) {
	d, err := s.repo.ManagedDrive(ctx, userID, s.slug)
	if err != nil {
		return File{}, nil, err
	}
	f, err := s.repo.ManagerFile(ctx, d.ID, nodeID)
	if err != nil {
		return f, nil, err
	}
	src, err := s.storage.Open(ctx, f.StorageKey)
	return f, src, err
}
func (s *ManagerService) Rename(ctx context.Context, userID, nodeID uuid.UUID, name string) error {
	name = managerSafeName(name)
	if name == "" {
		return ErrManagerInput
	}
	d, err := s.repo.ManagedDrive(ctx, userID, s.slug)
	if err != nil {
		return err
	}
	return s.repo.ManagerRename(ctx, d.ID, userID, nodeID, name)
}
func (s *ManagerService) Trash(ctx context.Context, userID, nodeID uuid.UUID) error {
	d, err := s.repo.ManagedDrive(ctx, userID, s.slug)
	if err != nil {
		return err
	}
	return s.repo.ManagerTrash(ctx, d.ID, userID, nodeID)
}
func managerSafeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." || len(name) > 255 || strings.ContainsAny(name, "/\\") {
		return ""
	}
	return filepath.Base(name)
}
func managerBlocked(name, mimeType string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".exe", ".dll", ".bat", ".cmd", ".com", ".ps1", ".sh", ".msi", ".scr", ".js", ".jar":
		return true
	}
	return mimeType == "application/x-msdownload" || mimeType == "application/x-dosexec"
}

type ManagerHandler struct {
	service   *ManagerService
	maxUpload int64
}

func NewManagerHandler(s *ManagerService, max int64) *ManagerHandler {
	return &ManagerHandler{service: s, maxUpload: max}
}
func (h *ManagerHandler) Index(c *gin.Context) {
	session := managerSession(c)
	parent, ok := managerOptionalUUID(c.Query("folder"))
	if !ok {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	page, err := h.service.Browse(c.Request.Context(), session.UserID, parent, c.Query("q"), c.Query("sort"))
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"Title": "Drive Publik tidak ditemukan", "Message": "Anda bukan anggota organisasi ini atau drive belum tersedia."})
		return
	}
	templateName := "public-drive-viewer.html"
	if session.Permissions["public_drive.write"] {
		templateName = "public-drive-manager.html"
	}
	c.HTML(http.StatusOK, templateName, gin.H{"Title": "Drive Publik", "Session": session, "Page": page, "Query": c.Query("q"), "Sort": c.Query("sort"), "MaxUpload": h.maxUpload})
}
func (h *ManagerHandler) CreateFolder(c *gin.Context) {
	session := managerSession(c)
	parent, ok := managerOptionalUUID(c.PostForm("parent_id"))
	if !ok {
		managerError(c, http.StatusBadRequest, "folder tujuan tidak valid")
		return
	}
	if err := h.service.CreateFolder(c.Request.Context(), session.UserID, parent, c.PostForm("name")); err != nil {
		managerHandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Folder publik berhasil dibuat"})
}
func (h *ManagerHandler) Upload(c *gin.Context) {
	session := managerSession(c)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxUpload+1024*1024)
	reader, err := c.Request.MultipartReader()
	if err != nil {
		managerError(c, 400, "form upload tidak valid")
		return
	}
	var parentRaw string
	uploaded := false
	for {
		part, e := reader.NextPart()
		if errors.Is(e, io.EOF) {
			break
		}
		if e != nil {
			managerError(c, 400, "upload terputus")
			return
		}
		if part.FormName() == "parent_id" {
			b, _ := io.ReadAll(io.LimitReader(part, 64))
			parentRaw = string(b)
			part.Close()
			continue
		}
		if part.FormName() != "file" || part.FileName() == "" {
			part.Close()
			continue
		}
		parent, ok := managerOptionalUUID(parentRaw)
		if !ok {
			part.Close()
			managerError(c, 400, "folder tujuan tidak valid")
			return
		}
		err = h.service.Upload(c.Request.Context(), session.UserID, parent, part.FileName(), part)
		part.Close()
		if err != nil {
			managerHandleError(c, err)
			return
		}
		uploaded = true
		break
	}
	if !uploaded {
		managerError(c, 400, "file wajib dipilih")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "File publik berhasil diunggah"})
}
func (h *ManagerHandler) Download(c *gin.Context) { h.serve(c, false) }
func (h *ManagerHandler) Preview(c *gin.Context)  { h.serve(c, true) }
func (h *ManagerHandler) serve(c *gin.Context, inline bool) {
	session := managerSession(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatus(400)
		return
	}
	f, src, err := h.service.File(c.Request.Context(), session.UserID, id)
	if err != nil {
		c.AbortWithStatus(404)
		return
	}
	defer src.Close()
	if inline && !managerPreviewable(f.MIME) {
		c.AbortWithStatus(415)
		return
	}
	kind := "attachment"
	if inline {
		kind = "inline"
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'self'")
		c.Header("Cache-Control", "private, max-age=300")
	}
	c.Header("Content-Disposition", mime.FormatMediaType(kind, map[string]string{"filename": f.Name}))
	c.Header("Content-Type", f.MIME)
	http.ServeContent(c.Writer, c.Request, f.Name, f.UpdatedAt, src)
}
func (h *ManagerHandler) Rename(c *gin.Context) {
	session := managerSession(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		managerError(c, 400, "item tidak valid")
		return
	}
	if err = h.service.Rename(c.Request.Context(), session.UserID, id, c.PostForm("name")); err != nil {
		managerHandleError(c, err)
		return
	}
	c.JSON(200, gin.H{"message": "Nama berhasil diubah"})
}
func (h *ManagerHandler) Trash(c *gin.Context) {
	session := managerSession(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		managerError(c, 400, "item tidak valid")
		return
	}
	if err = h.service.Trash(c.Request.Context(), session.UserID, id); err != nil {
		managerHandleError(c, err)
		return
	}
	c.JSON(200, gin.H{"message": "Dipindahkan ke Sampah"})
}
func managerSession(c *gin.Context) auth.Session {
	v, _ := c.Get(middleware.SessionKey)
	return v.(auth.Session)
}
func managerOptionalUUID(raw string) (*uuid.UUID, bool) {
	if raw == "" {
		return nil, true
	}
	id, err := uuid.Parse(raw)
	return &id, err == nil
}
func managerHandleError(c *gin.Context, err error) {
	if errors.Is(err, ErrManagerInput) {
		managerError(c, 400, "input atau tipe file tidak valid")
	} else if errors.Is(err, ErrManagerNotFound) {
		managerError(c, 404, "resource tidak ditemukan")
	} else {
		managerError(c, 409, "operasi gagal; nama mungkin sudah digunakan")
	}
}
func managerError(c *gin.Context, status int, msg string) { c.JSON(status, gin.H{"error": msg}) }
func managerPreviewable(t string) bool {
	return t == "application/pdf" || t == "image/jpeg" || t == "image/png" || t == "image/webp" || t == "video/mp4" || strings.HasPrefix(t, "text/plain")
}
