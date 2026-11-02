package mydrive

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"time"
)

var ErrNotFound = errors.New("resource not found")

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) PersonalDrive(ctx context.Context, userID uuid.UUID) (Drive, error) {
	var d Drive
	err := r.db.QueryRowContext(ctx, `SELECT id,name FROM drive_drives WHERE owner_user_id=? AND type='personal'`, userID.String()).Scan(&d.ID, &d.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return d, ErrNotFound
	}
	return d, err
}
func (r *Repository) Usage(ctx context.Context, driveID uuid.UUID) (int64, error) {
	var bytes int64
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(n.size_bytes),0) FROM drive_nodes n JOIN storage_objects o ON o.id=n.storage_object_id AND o.state='ready' WHERE n.drive_id=? AND n.type='file'`, driveID.String()).Scan(&bytes)
	return bytes, err
}
func (r *Repository) Node(ctx context.Context, driveID, nodeID uuid.UUID) (Node, error) {
	var n Node
	var parent sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT id,parent_id,name,type,COALESCE(mime_type,''),size_bytes,updated_at FROM drive_nodes WHERE id=? AND drive_id=? AND deleted_at IS NULL`, nodeID.String(), driveID.String()).Scan(&n.ID, &parent, &n.Name, &n.Type, &n.MIME, &n.Size, &n.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return n, ErrNotFound
	}
	if parent.Valid {
		id, parseErr := uuid.Parse(parent.String)
		if parseErr != nil {
			return n, parseErr
		}
		n.ParentID = &id
	}
	return n, err
}
func (r *Repository) List(ctx context.Context, driveID uuid.UUID, parentID *uuid.UUID, q, sort string) ([]Node, error) {
	order := "type DESC, LOWER(name)"
	if sort == "date" {
		order = "updated_at DESC"
	} else if sort == "size" {
		order = "size_bytes DESC"
	}
	var parent any
	if parentID != nil {
		parent = parentID.String()
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,parent_id,name,type,COALESCE(mime_type,''),size_bytes,updated_at FROM drive_nodes WHERE drive_id=? AND parent_id <=> ? AND deleted_at IS NULL AND (?='' OR name LIKE CONCAT('%',?,'%')) ORDER BY `+order, driveID.String(), parent, q, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	nodes := []Node{}
	for rows.Next() {
		var n Node
		var p sql.NullString
		if err = rows.Scan(&n.ID, &p, &n.Name, &n.Type, &n.MIME, &n.Size, &n.UpdatedAt); err != nil {
			return nil, err
		}
		if p.Valid {
			id, _ := uuid.Parse(p.String)
			n.ParentID = &id
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}
func (r *Repository) CreateFolder(ctx context.Context, driveID, userID uuid.UUID, parentID *uuid.UUID, name string) error {
	var parent any
	if parentID != nil {
		parent = parentID.String()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO drive_nodes(id,drive_id,parent_id,name,type,status,created_by,updated_by) VALUES(?,?,?,?, 'folder','ready',?,?)`, uuid.NewString(), driveID.String(), parent, name, userID.String(), userID.String())
	return err
}
func (r *Repository) CreateFile(ctx context.Context, driveID, userID uuid.UUID, parentID *uuid.UUID, name, mime, key, checksum string, size int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	objectID, nodeID := uuid.NewString(), uuid.NewString()
	if _, err = tx.ExecContext(ctx, `INSERT INTO storage_objects(id,storage_key,size_bytes,checksum_sha256,state) VALUES(?,?,?,?,'ready')`, objectID, key, size, checksum); err != nil {
		return err
	}
	var parent any
	if parentID != nil {
		parent = parentID.String()
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO drive_nodes(id,drive_id,parent_id,name,type,mime_type,size_bytes,storage_object_id,status,created_by,updated_by) VALUES(?,?,?,?, 'file',?,?,?,'ready',?,?)`, nodeID, driveID.String(), parent, name, mime, size, objectID, userID.String(), userID.String()); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO drive_file_versions(id,node_id,version_no,object_id,size_bytes,checksum_sha256,created_by) VALUES(?,?,1,?,?,?,?)`, uuid.NewString(), nodeID, objectID, size, checksum, userID.String()); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) Download(ctx context.Context, driveID, nodeID uuid.UUID) (Download, error) {
	var d Download
	var parent sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT n.id,n.parent_id,n.name,n.type,COALESCE(n.mime_type,''),n.size_bytes,n.updated_at,o.storage_key FROM drive_nodes n JOIN storage_objects o ON o.id=n.storage_object_id WHERE n.id=? AND n.drive_id=? AND n.type='file' AND n.deleted_at IS NULL AND n.status='ready' AND o.state='ready'`, nodeID.String(), driveID.String()).Scan(&d.Node.ID, &parent, &d.Node.Name, &d.Node.Type, &d.Node.MIME, &d.Node.Size, &d.Node.UpdatedAt, &d.StorageKey)
	if errors.Is(err, sql.ErrNoRows) {
		return d, ErrNotFound
	}
	return d, err
}
func (r *Repository) Rename(ctx context.Context, driveID, userID, nodeID uuid.UUID, name string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE drive_nodes SET name=?,updated_by=? WHERE id=? AND drive_id=? AND deleted_at IS NULL`, name, userID.String(), nodeID.String(), driveID.String())
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
func (r *Repository) Trash(ctx context.Context, driveID, userID, nodeID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `UPDATE drive_nodes SET deleted_at=UTC_TIMESTAMP(6),updated_by=? WHERE id=? AND drive_id=? AND deleted_at IS NULL`, userID.String(), nodeID.String(), driveID.String())
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
func (r *Repository) CreateShareLink(ctx context.Context, nodeID, userID uuid.UUID, tokenHash []byte, expiresAt *time.Time, allowDownload bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE sharing_share_links SET active=0 WHERE node_id=? AND active=1`, nodeID.String()); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO sharing_share_links(id,node_id,token_hash,expires_at,allow_download,active,created_by) VALUES(?,?,?, ?,?,1,?)`, uuid.NewString(), nodeID.String(), tokenHash, expiresAt, allowDownload, userID.String())
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) RevokeShareLink(ctx context.Context, driveID, nodeID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `UPDATE sharing_share_links l JOIN drive_nodes n ON n.id=l.node_id SET l.active=0 WHERE l.node_id=? AND n.drive_id=? AND l.active=1`, nodeID.String(), driveID.String())
	if err != nil {
		return err
	}
	if count, _ := res.RowsAffected(); count == 0 {
		return ErrNotFound
	}
	return nil
}
func (r *Repository) ShareStatuses(ctx context.Context, driveID uuid.UUID) ([]ShareStatus, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT n.id,l.active,l.expires_at FROM drive_nodes n LEFT JOIN sharing_share_links l ON l.id=(SELECT x.id FROM sharing_share_links x WHERE x.node_id=n.id ORDER BY x.created_at DESC LIMIT 1) WHERE n.drive_id=? AND n.deleted_at IS NULL`, driveID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	statuses := []ShareStatus{}
	now := time.Now().UTC()
	for rows.Next() {
		var status ShareStatus
		var active sql.NullBool
		var expires sql.NullTime
		if err = rows.Scan(&status.NodeID, &active, &expires); err != nil {
			return nil, err
		}
		status.Status = "inactive"
		if active.Valid && active.Bool {
			if expires.Valid && !expires.Time.After(now) {
				status.Status = "expired"
			} else {
				status.Status = "active"
			}
		}
		if expires.Valid {
			value := expires.Time.UTC()
			status.ExpiresAt = &value
		}
		statuses = append(statuses, status)
	}
	return statuses, rows.Err()
}
func (r *Repository) SharedRoot(ctx context.Context, tokenHash []byte) (SharedRoot, error) {
	var root SharedRoot
	var parent sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT n.id,n.parent_id,n.name,n.type,COALESCE(n.mime_type,''),n.size_bytes,n.updated_at,n.drive_id,u.display_name,l.allow_download FROM sharing_share_links l JOIN drive_nodes n ON n.id=l.node_id JOIN drive_drives d ON d.id=n.drive_id AND d.type='personal' JOIN auth_users u ON u.id=d.owner_user_id WHERE l.token_hash=? AND l.active=1 AND (l.expires_at IS NULL OR l.expires_at>UTC_TIMESTAMP(6)) AND n.deleted_at IS NULL AND n.status='ready'`, tokenHash).Scan(&root.ID, &parent, &root.Name, &root.Type, &root.MIME, &root.Size, &root.UpdatedAt, &root.DriveID, &root.OwnerName, &root.AllowDownload)
	if errors.Is(err, sql.ErrNoRows) {
		return root, ErrNotFound
	}
	if parent.Valid {
		id, _ := uuid.Parse(parent.String)
		root.ParentID = &id
	}
	if err == nil {
		_, _ = r.db.ExecContext(ctx, `UPDATE sharing_share_links SET last_accessed_at=UTC_TIMESTAMP(6) WHERE token_hash=?`, tokenHash)
	}
	return root, err
}
func (r *Repository) SharedFolder(ctx context.Context, root SharedRoot, folderID uuid.UUID) (Node, error) {
	var node Node
	var parent sql.NullString
	err := r.db.QueryRowContext(ctx, `WITH RECURSIVE tree AS (SELECT id,parent_id,name,type,mime_type,size_bytes,updated_at FROM drive_nodes WHERE id=? AND drive_id=? AND deleted_at IS NULL AND status='ready' UNION ALL SELECT n.id,n.parent_id,n.name,n.type,n.mime_type,n.size_bytes,n.updated_at FROM drive_nodes n JOIN tree p ON p.id=n.parent_id WHERE n.drive_id=? AND n.deleted_at IS NULL AND n.status='ready') SELECT id,parent_id,name,type,COALESCE(mime_type,''),size_bytes,updated_at FROM tree WHERE id=? AND type='folder'`, root.ID.String(), root.DriveID.String(), root.DriveID.String(), folderID.String()).Scan(&node.ID, &parent, &node.Name, &node.Type, &node.MIME, &node.Size, &node.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return node, ErrNotFound
	}
	if parent.Valid {
		id, _ := uuid.Parse(parent.String)
		node.ParentID = &id
	}
	return node, err
}
func (r *Repository) SharedChildren(ctx context.Context, root SharedRoot, folderID uuid.UUID) ([]Node, error) {
	if _, err := r.SharedFolder(ctx, root, folderID); err != nil {
		return nil, err
	}
	return r.List(ctx, root.DriveID, &folderID, "", "name")
}
func (r *Repository) SharedFile(ctx context.Context, root SharedRoot, nodeID uuid.UUID) (Download, error) {
	var file Download
	var parent sql.NullString
	err := r.db.QueryRowContext(ctx, `WITH RECURSIVE tree AS (SELECT id,parent_id FROM drive_nodes WHERE id=? AND drive_id=? AND deleted_at IS NULL AND status='ready' UNION ALL SELECT n.id,n.parent_id FROM drive_nodes n JOIN tree p ON p.id=n.parent_id WHERE n.drive_id=? AND n.deleted_at IS NULL AND n.status='ready') SELECT n.id,n.parent_id,n.name,n.type,COALESCE(n.mime_type,''),n.size_bytes,n.updated_at,o.storage_key FROM tree t JOIN drive_nodes n ON n.id=t.id JOIN storage_objects o ON o.id=n.storage_object_id AND o.state='ready' WHERE n.id=? AND n.type='file'`, root.ID.String(), root.DriveID.String(), root.DriveID.String(), nodeID.String()).Scan(&file.Node.ID, &parent, &file.Node.Name, &file.Node.Type, &file.Node.MIME, &file.Node.Size, &file.Node.UpdatedAt, &file.StorageKey)
	if errors.Is(err, sql.ErrNoRows) {
		return file, ErrNotFound
	}
	return file, err
}
