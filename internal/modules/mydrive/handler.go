package mydrive

import (
	"cloud-drive/internal/middleware"
	"cloud-drive/internal/modules/auth"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"mime"
	"net/http"
	"strconv"
)

type Handler struct {
	service   *Service
	maxUpload int64
}

func NewHandler(s *Service, max int64) *Handler { return &Handler{service: s, maxUpload: max} }
func (h *Handler) Index(c *gin.Context) {
	session := currentSession(c)
	parent, ok := optionalUUID(c.Query("folder"))
	if !ok {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	page, err := h.service.Browse(c.Request.Context(), session.UserID, parent, c.Query("q"), c.Query("sort"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		}
		c.HTML(status, "error.html", gin.H{"Title": "Folder tidak ditemukan", "Message": "Folder tidak tersedia atau bukan milik Anda."})
		return
	}
	c.HTML(http.StatusOK, "my-drive.html", gin.H{"Title": "Drive Saya", "Session": session, "Page": page, "Query": c.Query("q"), "Sort": c.Query("sort"), "MaxUpload": h.maxUpload})
}
func (h *Handler) Usage(c *gin.Context) {
	bytes, err := h.service.Usage(c.Request.Context(), currentSession(c).UserID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "pemakaian Drive Saya tidak tersedia"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bytes": bytes})
}
func (h *Handler) CreateFolder(c *gin.Context) {
	session := currentSession(c)
	parent, ok := optionalUUID(c.PostForm("parent_id"))
	if !ok {
		apiError(c, http.StatusBadRequest, "folder tujuan tidak valid")
		return
	}
	if err := h.service.CreateFolder(c.Request.Context(), session.UserID, parent, c.PostForm("name")); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Folder berhasil dibuat"})
}
func (h *Handler) Upload(c *gin.Context) {
	session := currentSession(c)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxUpload+1024*1024)
	reader, err := c.Request.MultipartReader()
	if err != nil {
		apiError(c, http.StatusBadRequest, "form upload tidak valid")
		return
	}
	var parentRaw string
	uploaded := false
	for {
		part, nextErr := reader.NextPart()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			apiError(c, http.StatusBadRequest, "upload terputus")
			return
		}
		if part.FormName() == "parent_id" {
			value, _ := io.ReadAll(io.LimitReader(part, 64))
			parentRaw = string(value)
			part.Close()
			continue
		}
		if part.FormName() != "file" || part.FileName() == "" {
			part.Close()
			continue
		}
		parent, ok := optionalUUID(parentRaw)
		if !ok {
			part.Close()
			apiError(c, http.StatusBadRequest, "folder tujuan tidak valid")
			return
		}
		err = h.service.Upload(c.Request.Context(), session.UserID, parent, part.FileName(), part)
		part.Close()
		if err != nil {
			handleError(c, err)
			return
		}
		uploaded = true
		break
	}
	if !uploaded {
		apiError(c, http.StatusBadRequest, "file wajib dipilih")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "File berhasil diunggah"})
}
func (h *Handler) Download(c *gin.Context) {
	h.serveFile(c, false)
}

func (h *Handler) Preview(c *gin.Context) {
	h.serveFile(c, true)
}

func (h *Handler) serveFile(c *gin.Context, inline bool) {
	session := currentSession(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	item, file, err := h.service.Download(c.Request.Context(), session.UserID, id)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	defer file.Close()
	if inline && !previewable(item.Node.MIME) {
		c.AbortWithStatusJSON(http.StatusUnsupportedMediaType, gin.H{"error": "preview tidak tersedia untuk tipe file ini"})
		return
	}
	if inline {
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'self'")
		c.Header("Cache-Control", "private, max-age=300")
	}
	dispositionType := "attachment"
	if inline {
		dispositionType = "inline"
	}
	disposition := mime.FormatMediaType(dispositionType, map[string]string{"filename": item.Node.Name})
	c.Header("Content-Disposition", disposition)
	c.Header("Content-Type", item.Node.MIME)
	c.Header("X-Content-Type-Options", "nosniff")
	http.ServeContent(c.Writer, c.Request, item.Node.Name, item.Node.UpdatedAt, file)
}

func previewable(mimeType string) bool {
	return mimeType == "application/pdf" || mimeType == "image/jpeg" || mimeType == "image/png" || mimeType == "image/webp" || mimeType == "video/mp4" || mimeType == "text/plain; charset=utf-8" || mimeType == "text/plain"
}
func (h *Handler) Rename(c *gin.Context) {
	session := currentSession(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apiError(c, http.StatusBadRequest, "file tidak valid")
		return
	}
	if err = h.service.Rename(c.Request.Context(), session.UserID, id, c.PostForm("name")); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Nama berhasil diubah"})
}
func (h *Handler) Trash(c *gin.Context) {
	session := currentSession(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apiError(c, http.StatusBadRequest, "file tidak valid")
		return
	}
	if err = h.service.Trash(c.Request.Context(), session.UserID, id); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Dipindahkan ke Sampah"})
}
func (h *Handler) CreateShare(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apiError(c, http.StatusBadRequest, "item tidak valid")
		return
	}
	days, err := strconv.Atoi(c.PostForm("expires_days"))
	if err != nil || days < 0 {
		apiError(c, http.StatusBadRequest, "masa berlaku tidak valid")
		return
	}
	allowDownload := c.PostForm("allow_download") != "false"
	share, err := h.service.CreateShare(c.Request.Context(), currentSession(c).UserID, id, days, allowDownload)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"token": share.Token, "allow_download": share.AllowDownload, "expires_at": share.ExpiresAt})
}
func (h *Handler) RevokeShare(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err == nil {
		err = h.service.RevokeShare(c.Request.Context(), currentSession(c).UserID, id)
	}
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Link publik dinonaktifkan"})
}
func (h *Handler) ShareStatuses(c *gin.Context) {
	statuses, err := h.service.ShareStatuses(c.Request.Context(), currentSession(c).UserID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "status link tidak tersedia"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": statuses})
}
func (h *Handler) SharedPage(c *gin.Context) {
	root, err := h.service.ResolveShare(c.Request.Context(), c.Param("token"))
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"Title": "Link tidak tersedia", "Message": "Link mungkin sudah dinonaktifkan, kedaluwarsa, atau item telah dipindahkan ke Sampah."})
		return
	}
	folderID, ok := optionalUUID(c.Query("folder"))
	if !ok {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	current, nodes, err := h.service.SharedBrowse(c.Request.Context(), root, folderID)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.HTML(http.StatusOK, "personal-share.html", gin.H{"Title": root.Name, "Root": root, "Current": current, "Nodes": nodes, "Token": c.Param("token")})
}
func (h *Handler) SharedPreview(c *gin.Context)  { h.serveSharedFile(c, true) }
func (h *Handler) SharedDownload(c *gin.Context) { h.serveSharedFile(c, false) }
func (h *Handler) serveSharedFile(c *gin.Context, inline bool) {
	root, err := h.service.ResolveShare(c.Request.Context(), c.Param("token"))
	if err != nil || (!inline && !root.AllowDownload) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	item, file, err := h.service.SharedDownload(c.Request.Context(), root, id)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	defer file.Close()
	if inline && !previewable(item.Node.MIME) {
		c.AbortWithStatus(http.StatusUnsupportedMediaType)
		return
	}
	kind := "attachment"
	if inline {
		kind = "inline"
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'self'")
	}
	c.Header("Content-Disposition", mime.FormatMediaType(kind, map[string]string{"filename": item.Node.Name}))
	c.Header("Content-Type", item.Node.MIME)
	c.Header("X-Content-Type-Options", "nosniff")
	http.ServeContent(c.Writer, c.Request, item.Node.Name, item.Node.UpdatedAt, file)
}
func currentSession(c *gin.Context) auth.Session {
	v, _ := c.Get(middleware.SessionKey)
	return v.(auth.Session)
}
func optionalUUID(raw string) (*uuid.UUID, bool) {
	if raw == "" {
		return nil, true
	}
	id, err := uuid.Parse(raw)
	return &id, err == nil
}
func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		apiError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotFound):
		apiError(c, http.StatusNotFound, "resource tidak ditemukan")
	default:
		apiError(c, http.StatusConflict, "operasi gagal; nama mungkin sudah digunakan")
	}
}
func apiError(c *gin.Context, status int, message string) { c.JSON(status, gin.H{"error": message}) }
