package main

import (
	"cloud-drive/internal/config"
	"cloud-drive/internal/database"
	"cloud-drive/internal/modules/auth"
	"context"
	"log"
	"os"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	email, password := os.Getenv("SUPERADMIN_EMAIL"), os.Getenv("SUPERADMIN_PASSWORD")
	if email == "" || password == "" {
		log.Fatal("SUPERADMIN_EMAIL and SUPERADMIN_PASSWORD are required")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Open(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	if _, err = conn.ExecContext(context.Background(), `CALL bootstrap_superadmin(?,?,?,@user_id)`, email, hash, "Super Admin"); err != nil {
		log.Fatal(err)
	}
	var id string
	if err = conn.QueryRowContext(context.Background(), `SELECT @user_id`).Scan(&id); err != nil {
		log.Fatal(err)
	}
	log.Printf("super admin ready: %s (%s)", email, id)
}
