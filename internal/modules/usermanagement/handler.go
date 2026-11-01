package usermanagement

import (
	"errors"
	"net/http"
	"strconv"

	"cloud-drive/internal/middleware"
	"cloud-drive/internal/modules/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func (h *Handler) Access(c *gin.Context) {
	s := adminSession(c)
	c.JSON(http.StatusOK, gin.H{"allowed": s.Permissions["users.manage"]})
}
func (h *Handler) Index(c *gin.Context) {
	users, err := h.service.List(c.Request.Context())
	if err != nil {
		c.HTML(500, "error.html", gin.H{"Title": "User Management tidak tersedia", "Message": "Data pengguna gagal dimuat."})
		return
	}
	c.HTML(200, "users.html", gin.H{"Title": "User Management", "Session": adminSession(c), "Users": users})
}
func (h *Handler) Create(c *gin.Context) {
	err := h.service.Create(c.Request.Context(), CreateInput{Email: c.PostForm("email"), DisplayName: c.PostForm("display_name"), Password: c.PostForm("password"), Role: c.PostForm("role")})
	h.respond(c, err, "Pengguna berhasil dibuat")
}
func (h *Handler) SetActive(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.respond(c, ErrInvalidInput, "")
		return
	}
	active, err := strconv.ParseBool(c.PostForm("active"))
	if err == nil {
		err = h.service.SetActive(c.Request.Context(), adminSession(c).UserID, id, active)
	}
	h.respond(c, err, "Status pengguna berhasil diperbarui")
}
func (h *Handler) ResetPassword(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err == nil {
		err = h.service.ResetPassword(c.Request.Context(), id, c.PostForm("password"))
	}
	h.respond(c, err, "Password berhasil diubah")
}
func (h *Handler) SetRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err == nil {
		err = h.service.SetRole(c.Request.Context(), adminSession(c).UserID, id, c.PostForm("role"))
	}
	h.respond(c, err, "Peran pengguna berhasil diperbarui")
}
func (h *Handler) respond(c *gin.Context, err error, message string) {
	if err == nil {
		c.JSON(200, gin.H{"message": message})
		return
	}
	status := 400
	if errors.Is(err, ErrNotFound) {
		status = 404
	} else if errors.Is(err, ErrEmailExists) {
		status = 409
	} else if errors.Is(err, ErrSelfAction) {
		status = 409
	}
	c.JSON(status, gin.H{"error": err.Error()})
}
func adminSession(c *gin.Context) auth.Session {
	v, _ := c.Get(middleware.SessionKey)
	return v.(auth.Session)
}
