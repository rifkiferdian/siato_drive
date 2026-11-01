package auth

import (
	"cloud-drive/internal/config"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Handler struct {
	service *Service
	cfg     config.Config
}

func (h *Handler) Profile(c *gin.Context) {
	session := c.MustGet("session").(Session)
	c.HTML(http.StatusOK, "profile.html", gin.H{"Title": "Profil Saya", "Session": session})
}
func (h *Handler) UpdateProfile(c *gin.Context) {
	session := c.MustGet("session").(Session)
	if err := h.service.UpdateDisplayName(c.Request.Context(), session.UserID, c.PostForm("display_name")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Profil berhasil diperbarui"})
}
func (h *Handler) ChangePassword(c *gin.Context) {
	session := c.MustGet("session").(Session)
	if err := h.service.ChangePassword(c.Request.Context(), session, c.PostForm("current_password"), c.PostForm("new_password")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Password berhasil diubah"})
}

func NewHandler(s *Service, cfg config.Config) *Handler { return &Handler{service: s, cfg: cfg} }
func (h *Handler) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{"Title": "Masuk"})
}
func (h *Handler) Login(c *gin.Context) {
	token, _, err := h.service.Login(c.Request.Context(), c.PostForm("email"), c.PostForm("password"), c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"Title": "Masuk", "Error": ErrInvalidCredentials.Error(), "Email": c.PostForm("email")})
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{Name: h.cfg.CookieName, Value: token, Path: "/", MaxAge: int(h.cfg.SessionTTL.Seconds()), HttpOnly: true, Secure: h.cfg.CookieSecure, SameSite: http.SameSiteLaxMode})
	c.Redirect(http.StatusSeeOther, "/dashboard")
}
func (h *Handler) Logout(c *gin.Context) {
	token, _ := c.Cookie(h.cfg.CookieName)
	_ = h.service.Logout(c.Request.Context(), token)
	http.SetCookie(c.Writer, &http.Cookie{Name: h.cfg.CookieName, Path: "/", MaxAge: -1, HttpOnly: true, Secure: h.cfg.CookieSecure, SameSite: http.SameSiteLaxMode})
	c.Redirect(http.StatusSeeOther, "/login")
}
