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
