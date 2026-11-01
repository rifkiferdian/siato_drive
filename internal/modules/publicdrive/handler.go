package publicdrive

import (
	"cloud-drive/internal/platform/storage"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	if raw := c.Query("folder"); raw != "" {
		id, e := uuid.Parse(raw)
		if e != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		parent = &id
	}
	nodes, err := h.repo.List(c.Request.Context(), o.ID, parent, c.Query("q"), c.Query("type"), c.Query("sort"))
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.HTML(http.StatusOK, "public.html", gin.H{"Title": o.Name + " — Drive Publik", "Org": o, "Nodes": nodes, "Parent": parent, "Query": c.Query("q")})
}
func (h *Handler) Download(c *gin.Context) {
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
	h.repo.LogDownload(c.Request.Context(), o.ID, id, c.ClientIP())
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, strings.ReplaceAll(f.Name, "%", "%25")))
	c.Header("Content-Type", f.MIME)
	c.Header("X-Content-Type-Options", "nosniff")
	http.ServeContent(c.Writer, c.Request, f.Name, f.UpdatedAt, src)
}
