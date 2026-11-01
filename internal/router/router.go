package router

import (
	"cloud-drive/internal/config"
	"cloud-drive/internal/middleware"
	"cloud-drive/internal/modules/auth"
	"cloud-drive/internal/modules/mydrive"
	"cloud-drive/internal/modules/publicdrive"
	"github.com/gin-gonic/gin"
	"html/template"
	"net/http"
)

func New(cfg config.Config, templates *template.Template, ah *auth.Handler, as *auth.Service, ph *publicdrive.Handler, mh *mydrive.Handler) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery(), middleware.SecurityHeaders())
	r.SetHTMLTemplate(templates)
	r.StaticFS("/static", http.Dir("web/static"))
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusSeeOther, "/login") })
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/login", ah.LoginPage)
	r.POST("/login", ah.Login)
	r.GET("/public", ph.Index)
	r.GET("/public/:organizationSlug", ph.Index)
	r.GET("/public/:organizationSlug/folders/:id", func(c *gin.Context) { c.Request.URL.RawQuery = "folder=" + c.Param("id"); ph.Index(c) })
	r.GET("/public/:organizationSlug/files/:id/download", ph.Download)
	secured := r.Group("")
	secured.Use(middleware.RequireAuth(as, cfg.CookieName))
	secured.POST("/logout", middleware.RequireCSRF(), ah.Logout)
	secured.GET("/dashboard", func(c *gin.Context) {
		s, _ := c.Get(middleware.SessionKey)
		c.HTML(http.StatusOK, "dashboard.html", gin.H{"Title": "Dashboard", "Session": s.(auth.Session)})
	})
	secured.GET("/drive/my-drive", mh.Index)
	secured.POST("/api/v1/my-drive/folders", middleware.RequireCSRF(), mh.CreateFolder)
	secured.POST("/api/v1/my-drive/upload", middleware.RequireCSRF(), mh.Upload)
	secured.GET("/api/v1/my-drive/nodes/:id/download", mh.Download)
	secured.GET("/api/v1/my-drive/nodes/:id/preview", mh.Preview)
	secured.POST("/api/v1/my-drive/nodes/:id/rename", middleware.RequireCSRF(), mh.Rename)
	secured.POST("/api/v1/my-drive/nodes/:id/trash", middleware.RequireCSRF(), mh.Trash)
	secured.GET("/drive/shared", placeholder("Drive Bersama"))
	secured.GET("/drive/public", middleware.RequirePermission("public_drive.write"), placeholder("Kelola Drive Publik"))
	secured.GET("/drive/trash", placeholder("Sampah"))
	return r
}
func placeholder(title string) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, _ := c.Get(middleware.SessionKey)
		c.HTML(http.StatusOK, "dashboard.html", gin.H{"Title": title, "Session": s.(auth.Session), "Notice": "Modul ini telah diproteksi session dan disiapkan untuk operasi drive berikutnya."})
	}
}
