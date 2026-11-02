package publicdrive

import (
	"cloud-drive/internal/platform/storage"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"mime"
	"net/http"
	"strings"
)

type Handler struct {
	repo        *Repository
	storage     storage.Provider
	defaultSlug string
}

func NewHandler(r *Repository, s storage.Provider, slug string) *Handler {
	return &Handler{repo: r, storage: s, defaultSlug: slug}
}
func (h *Handler) Index(c *gin.Context) {
	slug := c.Param("organizationSlug")
	if slug == "" {
		slug = h.defaultSlug
	}
	o, err := h.repo.Organization(c.Request.Context(), slug)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"Title": "Tidak ditemukan", "Message": "Portal publik tidak ditemukan."})
		return
	}
	var parent *uuid.UUID
	var breadcrumbs []Crumb
	if raw := c.Query("folder"); raw != "" {
		id, e := uuid.Parse(raw)
		if e != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		parent = &id
		current, e := h.repo.PublicNode(c.Request.Context(), o.ID, id)
		if e != nil || current.Type != "folder" {
			c.HTML(http.StatusNotFound, "error.html", gin.H{"Title": "Folder tidak ditemukan", "Message": "Folder publik tidak ditemukan atau sudah tidak tersedia."})
			return
		}
		for depth := 0; depth < 100; depth++ {
			currentID := current.ID
			breadcrumbs = append([]Crumb{{ID: &currentID, Name: current.Name}}, breadcrumbs...)
			if current.ParentID == nil {
				break
			}
			current, e = h.repo.PublicNode(c.Request.Context(), o.ID, *current.ParentID)
			if e != nil {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}
		}
	}
	nodes, err := h.repo.List(c.Request.Context(), o.ID, parent, c.Query("q"), c.Query("type"), c.Query("sort"))
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.HTML(http.StatusOK, "public.html", gin.H{"Title": o.Name + " — Drive Publik", "Org": o, "Nodes": nodes, "Parent": parent, "Breadcrumbs": breadcrumbs, "Query": c.Query("q")})
}
func (h *Handler) Embed(c *gin.Context) {
	h.renderSharedFolder(c, true)
}
func (h *Handler) SharedFolder(c *gin.Context) {
	h.renderSharedFolder(c, false)
}
func (h *Handler) renderSharedFolder(c *gin.Context, allowFrame bool) {
	o, err := h.repo.Organization(c.Request.Context(), h.defaultSlug)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	folder, err := h.repo.PublicNode(c.Request.Context(), o.ID, id)
	if err != nil || folder.Type != "folder" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	breadcrumbs := []Crumb{}
	current := folder
	for depth := 0; depth < 100; depth++ {
		currentID := current.ID
		breadcrumbs = append([]Crumb{{ID: &currentID, Name: current.Name}}, breadcrumbs...)
		if current.ParentID == nil {
			break
		}
		current, err = h.repo.PublicNode(c.Request.Context(), o.ID, *current.ParentID)
		if err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
	}
	nodes, err := h.repo.List(c.Request.Context(), o.ID, &id, "", "", c.Query("sort"))
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if allowFrame {
		// Hanya endpoint embed yang diizinkan berada di iframe lintas situs.
		c.Writer.Header().Del("X-Frame-Options")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; frame-src 'self'; object-src 'none'; frame-ancestors *")
	}
	basePath := "/share/folders/"
	if allowFrame {
		basePath = "/embed/folders/"
	}
	c.HTML(http.StatusOK, "public-embed.html", gin.H{"Title": folder.Name, "Org": o, "Folder": folder, "Nodes": nodes, "Breadcrumbs": breadcrumbs, "BasePath": basePath, "Sort": c.Query("sort"), "Embed": allowFrame})
}
func (h *Handler) Download(c *gin.Context) {
	h.serveFile(c, false)
}
func (h *Handler) Preview(c *gin.Context) {
	h.serveFile(c, true)
}
func (h *Handler) serveFile(c *gin.Context, inline bool) {
	o, err := h.repo.Organization(c.Request.Context(), c.Param("organizationSlug"))
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	f, err := h.repo.PublicFile(c.Request.Context(), o.ID, id)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	src, err := h.storage.Open(c.Request.Context(), f.StorageKey)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	defer src.Close()
	if inline && !publicPreviewable(f.MIME) {
		c.AbortWithStatus(http.StatusUnsupportedMediaType)
		return
	}
	if !inline {
		h.repo.LogDownload(c.Request.Context(), o.ID, id, c.ClientIP())
	}
	disposition := "attachment"
	if inline {
		disposition = "inline"
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'self'")
		c.Header("Cache-Control", "public, max-age=300")
	}
	value := mime.FormatMediaType(disposition, map[string]string{"filename": f.Name})
	if value == "" {
		value = fmt.Sprintf(`%s; filename*=UTF-8''%s`, disposition, strings.ReplaceAll(f.Name, "%", "%25"))
	}
	c.Header("Content-Disposition", value)
	c.Header("Content-Type", f.MIME)
	c.Header("X-Content-Type-Options", "nosniff")
	http.ServeContent(c.Writer, c.Request, f.Name, f.UpdatedAt, src)
}

func publicPreviewable(mimeType string) bool {
	mimeType = strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
	switch mimeType {
	case "application/pdf", "image/jpeg", "image/png", "image/gif", "image/webp", "video/mp4", "text/plain":
		return true
	default:
		return false
	}
}
