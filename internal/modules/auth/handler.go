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
