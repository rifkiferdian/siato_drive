package trash

import (
	"cloud-drive/internal/middleware"
	"cloud-drive/internal/modules/auth"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }
func (h *Handler) Index(c *gin.Context) {
	session := trashSession(c)
	items, err := h.service.List(c.Request.Context(), session.UserID, session.Permissions["public_drive.write"], c.Query("q"), c.Query("type"))
	if err != nil {
		c.HTML(500, "error.html", gin.H{"Title": "Sampah tidak tersedia", "Message": "Data Sampah gagal dimuat."})
		return
	}
	c.HTML(200, "trash.html", gin.H{"Title": "Sampah", "Session": session, "Items": items, "Query": c.Query("q"), "Type": c.Query("type")})
}
func (h *Handler) Restore(c *gin.Context)         { h.mutate(c, false) }
func (h *Handler) PermanentDelete(c *gin.Context) { h.mutate(c, true) }
func (h *Handler) mutate(c *gin.Context, permanent bool) {
	session := trashSession(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "item tidak valid"})
		return
	}
	if permanent {
		err = h.service.PermanentDelete(c.Request.Context(), session.UserID, id, session.Permissions["public_drive.write"])
	} else {
		err = h.service.Restore(c.Request.Context(), session.UserID, id, session.Permissions["public_drive.write"])
	}
	if err != nil {
		status, msg := 409, "operasi gagal"
		if errors.Is(err, ErrNotFound) {
			status = 404
			msg = "item tidak ditemukan"
		} else if errors.Is(err, ErrParentDeleted) {
			msg = "pulihkan folder induk terlebih dahulu"
		}
		c.JSON(status, gin.H{"error": msg})
		return
	}
	msg := "Item berhasil dipulihkan"
	if permanent {
		msg = "Item dihapus permanen"
	}
	c.JSON(200, gin.H{"message": msg})
}
func trashSession(c *gin.Context) auth.Session {
	v, _ := c.Get(middleware.SessionKey)
	return v.(auth.Session)
}
