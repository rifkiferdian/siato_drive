package middleware

import (
	"cloud-drive/internal/modules/auth"
	"crypto/subtle"
	"github.com/gin-gonic/gin"
	"net/http"
)

const SessionKey = "session"

func RequireAuth(s *auth.Service, cookie string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cookie)
		if err != nil {
			redirectOrJSON(c, http.StatusUnauthorized, "/login", "authentication required")
			return
		}
		session, err := s.Authenticate(c.Request.Context(), token)
		if err != nil {
			redirectOrJSON(c, http.StatusUnauthorized, "/login", "authentication required")
			return
		}
		c.Set(SessionKey, session)
		c.Next()
	}
}
func RequirePermission(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get(SessionKey)
		if !ok || !v.(auth.Session).Permissions[code] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			return
		}
		c.Next()
	}
}
func RequireCSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get(SessionKey)
		if !ok {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		expected := v.(auth.Session).CSRFToken
		got := c.GetHeader("X-CSRF-Token")
		if got == "" {
			got = c.PostForm("csrf_token")
		}
		if len(got) != len(expected) || subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid CSRF token"})
			return
		}
		c.Next()
	}
}
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; media-src 'self'; object-src 'none'; frame-ancestors 'none'")
		c.Next()
	}
}
func redirectOrJSON(c *gin.Context, status int, path, msg string) {
	c.Abort()
	if c.GetHeader("Accept") == "application/json" || len(c.Request.URL.Path) >= 5 && c.Request.URL.Path[:5] == "/api/" {
		c.JSON(status, gin.H{"error": msg})
	} else {
		c.Redirect(http.StatusSeeOther, path)
	}
}
