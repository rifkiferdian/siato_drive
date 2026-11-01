package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"github.com/google/uuid"
	"time"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }
func (r *Repository) Credentials(ctx context.Context, email string) (User, string, error) {
	var u User
	var hash string
	err := r.db.QueryRowContext(ctx, `SELECT id,email,display_name,active,password_hash FROM auth_users WHERE lower(email)=lower(?)`, email).Scan(&u.ID, &u.Email, &u.DisplayName, &u.Active, &hash)
	return u, hash, err
}
func (r *Repository) CreateSession(ctx context.Context, userID uuid.UUID, token, csrf, ip, ua string, expires time.Time) error {
	h := sha256.Sum256([]byte(token))
	_, err := r.db.ExecContext(ctx, `INSERT INTO auth_sessions(id,token_hash,user_id,csrf_token,expires_at,ip,user_agent) VALUES(UUID(),?,?,?,?,NULLIF(?,''),?)`, h[:], userID.String(), csrf, expires.UTC(), ip, ua)
	return err
}
func (r *Repository) Session(ctx context.Context, token string) (Session, error) {
	h := sha256.Sum256([]byte(token))
	var s Session
	err := r.db.QueryRowContext(ctx, `SELECT s.id,s.user_id,s.csrf_token,s.expires_at,u.id,u.email,u.display_name,u.active FROM auth_sessions s JOIN auth_users u ON u.id=s.user_id WHERE s.token_hash=? AND s.expires_at>UTC_TIMESTAMP() AND u.active=1`, h[:]).Scan(&s.ID, &s.UserID, &s.CSRFToken, &s.ExpiresAt, &s.User.ID, &s.User.Email, &s.User.DisplayName, &s.User.Active)
	if err != nil {
		return s, err
	}
	s.Permissions = map[string]bool{}
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT p.code FROM auth_user_roles ur JOIN auth_role_permissions rp ON rp.role_id=ur.role_id JOIN auth_permissions p ON p.id=rp.permission_id WHERE ur.user_id=?`, s.UserID.String())
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var p string
		if err = rows.Scan(&p); err != nil {
			return s, err
		}
		s.Permissions[p] = true
	}
	return s, rows.Err()
}
func (r *Repository) DeleteSession(ctx context.Context, token string) error {
	h := sha256.Sum256([]byte(token))
	_, err := r.db.ExecContext(ctx, `DELETE FROM auth_sessions WHERE token_hash=?`, h[:])
	return err
}
func (r *Repository) UpdateDisplayName(ctx context.Context, userID uuid.UUID, name string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE auth_users SET display_name=? WHERE id=?`, name, userID.String())
	return err
}
func (r *Repository) UpdatePassword(ctx context.Context, userID, currentSessionID uuid.UUID, hash string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE auth_users SET password_hash=? WHERE id=?`, hash, userID.String()); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM auth_sessions WHERE user_id=? AND id<>?`, userID.String(), currentSessionID.String()); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) LogLogin(ctx context.Context, email string, userID *uuid.UUID, success bool, ip string) {
	var id any
	if userID != nil {
		id = userID.String()
	}
	_, _ = r.db.ExecContext(ctx, `INSERT INTO auth_login_logs(email,user_id,success,ip) VALUES(?,?,?,NULLIF(?,''))`, email, id, success, ip)
}
func IsNotFound(err error) bool { return err == sql.ErrNoRows }
