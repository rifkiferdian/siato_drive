package main

import (
	"cloud-drive/internal/config"
	"cloud-drive/internal/database"
	"cloud-drive/internal/modules/auth"
	"cloud-drive/internal/modules/mydrive"
	"cloud-drive/internal/modules/publicdrive"
	"cloud-drive/internal/modules/shareddrive"
	"cloud-drive/internal/modules/trash"
	"cloud-drive/internal/platform/storage"
	"cloud-drive/internal/router"
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	store, err := storage.NewLocal(cfg.StorageRoot)
	if err != nil {
		log.Fatal(err)
	}
	templates, err := template.ParseGlob("web/templates/*.html")
	if err != nil {
		log.Fatal(err)
	}
	ar := auth.NewRepository(db)
	as := auth.NewService(ar, cfg.SessionTTL)
	myDriveService := mydrive.NewService(mydrive.NewRepository(db), store, cfg.MaxUploadBytes)
	publicRepo := publicdrive.NewRepository(db)
	publicManager := publicdrive.NewManagerHandler(publicdrive.NewManagerService(publicRepo, store, cfg.DefaultOrgSlug, cfg.MaxUploadBytes), cfg.MaxUploadBytes)
	sharedManager := shareddrive.NewManagerHandler(shareddrive.NewManagerService(shareddrive.NewRepository(db), store, cfg.DefaultOrgSlug, cfg.MaxUploadBytes), cfg.MaxUploadBytes)
	trashHandler := trash.NewHandler(trash.NewService(trash.NewRepository(db), store))
	app := router.New(cfg, templates, auth.NewHandler(as, cfg), as, publicdrive.NewHandler(publicRepo, store, cfg.DefaultOrgSlug), mydrive.NewHandler(myDriveService, cfg.MaxUploadBytes), publicManager, sharedManager, trashHandler)
	srv := &http.Server{Addr: cfg.Addr, Handler: app, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 5 * time.Minute, IdleTimeout: 2 * time.Minute}
	go func() {
		log.Printf("cloud-drive listening on %s", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}
