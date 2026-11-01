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
	var organizationID string
	if _, err = conn.ExecContext(context.Background(), `INSERT INTO org_organizations(id,slug,name,active) VALUES(UUID(),?, 'Organisasi Default',1) ON DUPLICATE KEY UPDATE active=1`, cfg.DefaultOrgSlug); err != nil {
		log.Fatal(err)
	}
	if err = conn.QueryRowContext(context.Background(), `SELECT id FROM org_organizations WHERE slug=?`, cfg.DefaultOrgSlug).Scan(&organizationID); err != nil {
		log.Fatal(err)
	}
	if _, err = conn.ExecContext(context.Background(), `INSERT INTO org_memberships(organization_id,user_id,status) VALUES(?,?,'active') ON DUPLICATE KEY UPDATE status='active'`, organizationID, id); err != nil {
		log.Fatal(err)
	}
	if _, err = conn.ExecContext(context.Background(), `INSERT IGNORE INTO drive_drives(id,organization_id,name,type) VALUES(UUID(),?,'Drive Publik','public')`, organizationID); err != nil {
		log.Fatal(err)
	}
	if _, err = conn.ExecContext(context.Background(), `INSERT INTO drive_drive_members(drive_id,user_id,role) SELECT id,?,'manager' FROM drive_drives WHERE organization_id=? AND type='public' ON DUPLICATE KEY UPDATE role='manager'`, id, organizationID); err != nil {
		log.Fatal(err)
	}
	if _, err = conn.ExecContext(context.Background(), `INSERT INTO drive_drives(id,organization_id,name,type) SELECT UUID(),?,'Drive Bersama','shared' WHERE NOT EXISTS(SELECT 1 FROM drive_drives WHERE organization_id=? AND type='shared' AND name='Drive Bersama')`, organizationID, organizationID); err != nil {
		log.Fatal(err)
	}
	if _, err = conn.ExecContext(context.Background(), `INSERT INTO drive_drive_members(drive_id,user_id,role) SELECT id,?,'manager' FROM drive_drives WHERE organization_id=? AND type='shared' AND name='Drive Bersama' ON DUPLICATE KEY UPDATE role='manager'`, id, organizationID); err != nil {
		log.Fatal(err)
	}
	log.Printf("super admin ready: %s (%s)", email, id)
	log.Printf("public drive ready: /public/%s", cfg.DefaultOrgSlug)
	log.Printf("shared drive ready: Drive Bersama")
}
