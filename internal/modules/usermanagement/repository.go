package usermanagement

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("pengguna tidak ditemukan")
var ErrEmailExists = errors.New("email sudah digunakan")

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context) ([]User, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT u.id,u.email,u.display_name,u.active,u.created_at,CASE WHEN SUM(ar.code IN ('super_admin','org_admin'))>0 THEN 'admin' ELSE 'user' END role,COALESCE((SELECT SUM(n.size_bytes) FROM drive_drives d JOIN drive_nodes n ON n.drive_id=d.id AND n.type='file' JOIN storage_objects o ON o.id=n.storage_object_id AND o.state='ready' WHERE d.type='personal' AND d.owner_user_id=u.id),0) storage_bytes FROM auth_users u LEFT JOIN auth_user_roles ur ON ur.user_id=u.id LEFT JOIN auth_roles ar ON ar.id=ur.role_id GROUP BY u.id,u.email,u.display_name,u.active,u.created_at ORDER BY u.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []User{}
	for rows.Next() {
		var u User
		if err = rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Active, &u.CreatedAt, &u.Role, &u.StorageBytes); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *Repository) Create(ctx context.Context, in CreateInput, passwordHash, orgSlug string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var exists int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM auth_users WHERE LOWER(email)=LOWER(?)`, in.Email).Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		return ErrEmailExists
	}
	userID := uuid.NewString()
	if _, err = tx.ExecContext(ctx, `INSERT INTO auth_users(id,email,password_hash,display_name,active) VALUES(?,LOWER(?),?,?,1)`, userID, in.Email, passwordHash, in.DisplayName); err != nil {
		return err
	}
	roleCode := "member"
	if in.Role == "admin" {
		roleCode = "org_admin"
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO auth_user_roles(user_id,role_id) SELECT ?,id FROM auth_roles WHERE code=?`, userID, roleCode); err != nil {
		return err
	}
	if in.Role == "admin" {
		if _, err = tx.ExecContext(ctx, `INSERT IGNORE INTO auth_role_permissions(role_id,permission_id) SELECT r.id,p.id FROM auth_roles r JOIN auth_permissions p ON p.code IN ('users.manage','drives.manage','public_drive.write','files.read','files.write') WHERE r.code='org_admin'`); err != nil {
			return err
		}
	}
	var orgID string
	if err = tx.QueryRowContext(ctx, `SELECT id FROM org_organizations WHERE slug=? AND active=1`, orgSlug).Scan(&orgID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO org_memberships(organization_id,user_id,status) VALUES(?,?,'active')`, orgID, userID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO drive_drives(id,owner_user_id,name,type) VALUES(UUID(),?,'Drive Saya','personal')`, userID); err != nil {
		return err
	}
	sharedRole := "editor"
	if in.Role == "admin" {
		sharedRole = "manager"
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO drive_drive_members(drive_id,user_id,role) SELECT id,?,? FROM drive_drives WHERE organization_id=? AND type='shared'`, userID, sharedRole, orgID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	res, err := r.db.ExecContext(ctx, `UPDATE auth_users SET active=? WHERE id=?`, active, id.String())
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	if !active {
		_, _ = r.db.ExecContext(ctx, `DELETE FROM auth_sessions WHERE user_id=?`, id.String())
	}
	return nil
}

func (r *Repository) ResetPassword(ctx context.Context, id uuid.UUID, hash string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE auth_users SET password_hash=? WHERE id=?`, hash, id.String())
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	_, _ = r.db.ExecContext(ctx, `DELETE FROM auth_sessions WHERE user_id=?`, id.String())
	return nil
}

func (r *Repository) SetRole(ctx context.Context, id uuid.UUID, role, orgSlug string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var orgID string
	if err = tx.QueryRowContext(ctx, `SELECT id FROM org_organizations WHERE slug=? AND active=1`, orgSlug).Scan(&orgID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE ur FROM auth_user_roles ur JOIN auth_roles r ON r.id=ur.role_id WHERE ur.user_id=? AND r.code IN ('org_admin','member')`, id.String()); err != nil {
		return err
	}
	roleCode, driveRole := "member", "editor"
	if role == "admin" {
		roleCode, driveRole = "org_admin", "manager"
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO auth_user_roles(user_id,role_id) SELECT ?,id FROM auth_roles WHERE code=?`, id.String(), roleCode)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	if role == "admin" {
		_, err = tx.ExecContext(ctx, `INSERT IGNORE INTO auth_role_permissions(role_id,permission_id) SELECT r.id,p.id FROM auth_roles r JOIN auth_permissions p ON p.code IN ('users.manage','drives.manage','public_drive.write','files.read','files.write') WHERE r.code='org_admin'`)
		if err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE drive_drive_members dm JOIN drive_drives d ON d.id=dm.drive_id SET dm.role=? WHERE dm.user_id=? AND d.organization_id=? AND d.type='shared'`, driveRole, id.String(), orgID); err != nil {
		return err
	}
	return tx.Commit()
}
