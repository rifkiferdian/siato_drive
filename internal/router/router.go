package router

import (
	"cloud-drive/internal/config"
	"cloud-drive/internal/middleware"
	"cloud-drive/internal/modules/auth"
	"cloud-drive/internal/modules/mydrive"
	"cloud-drive/internal/modules/publicdrive"
	"cloud-drive/internal/modules/shareddrive"
	"cloud-drive/internal/modules/trash"
	"cloud-drive/internal/modules/usermanagement"
	"github.com/gin-gonic/gin"
	"html/template"
	"net/http"
)

func New(cfg config.Config, templates *template.Template, ah *auth.Handler, as *auth.Service, ph *publicdrive.Handler, mh *mydrive.Handler, pmh *publicdrive.ManagerHandler, smh *shareddrive.ManagerHandler, th *trash.Handler, uh *usermanagement.Handler) *gin.Engine {
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
	secured.GET("/profile", ah.Profile)
	secured.POST("/api/v1/profile", middleware.RequireCSRF(), ah.UpdateProfile)
	secured.POST("/api/v1/profile/password", middleware.RequireCSRF(), ah.ChangePassword)
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
	sharedManager := secured.Group("")
	sharedManager.Use(middleware.RequirePermission("files.write"))
	sharedManager.GET("/drive/shared", smh.Index)
	sharedManager.POST("/api/v1/shared-drive/folders", middleware.RequireCSRF(), smh.CreateFolder)
	sharedManager.POST("/api/v1/shared-drive/upload", middleware.RequireCSRF(), smh.Upload)
	sharedManager.GET("/api/v1/shared-drive/nodes/:id/download", smh.Download)
	sharedManager.GET("/api/v1/shared-drive/nodes/:id/preview", smh.Preview)
	sharedManager.POST("/api/v1/shared-drive/nodes/:id/rename", middleware.RequireCSRF(), smh.Rename)
	sharedManager.POST("/api/v1/shared-drive/nodes/:id/trash", middleware.RequireCSRF(), smh.Trash)
	publicManager := secured.Group("")
	publicManager.Use(middleware.RequirePermission("public_drive.write"))
	publicManager.GET("/drive/public", pmh.Index)
	publicManager.POST("/api/v1/public-drive/folders", middleware.RequireCSRF(), pmh.CreateFolder)
	publicManager.POST("/api/v1/public-drive/upload", middleware.RequireCSRF(), pmh.Upload)
	publicManager.GET("/api/v1/public-drive/nodes/:id/download", pmh.Download)
	publicManager.GET("/api/v1/public-drive/nodes/:id/preview", pmh.Preview)
	publicManager.POST("/api/v1/public-drive/nodes/:id/rename", middleware.RequireCSRF(), pmh.Rename)
	publicManager.POST("/api/v1/public-drive/nodes/:id/trash", middleware.RequireCSRF(), pmh.Trash)
	secured.GET("/drive/trash", th.Index)
	secured.POST("/api/v1/trash/nodes/:id/restore", middleware.RequireCSRF(), th.Restore)
	secured.POST("/api/v1/trash/nodes/:id/permanent-delete", middleware.RequireCSRF(), th.PermanentDelete)
	secured.GET("/api/v1/admin/access", uh.Access)
	admin := secured.Group("")
	admin.Use(middleware.RequirePermission("users.manage"))
	admin.GET("/admin/users", uh.Index)
	admin.POST("/api/v1/admin/users", middleware.RequireCSRF(), uh.Create)
	admin.POST("/api/v1/admin/users/:id/status", middleware.RequireCSRF(), uh.SetActive)
	admin.POST("/api/v1/admin/users/:id/password", middleware.RequireCSRF(), uh.ResetPassword)
	admin.POST("/api/v1/admin/users/:id/role", middleware.RequireCSRF(), uh.SetRole)
	return r
}
func placeholder(title string) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, _ := c.Get(middleware.SessionKey)
		c.HTML(http.StatusOK, "dashboard.html", gin.H{"Title": title, "Session": s.(auth.Session), "Notice": "Modul ini telah diproteksi session dan disiapkan untuk operasi drive berikutnya."})
	}
}
