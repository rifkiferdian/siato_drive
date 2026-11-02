package mydrive

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
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
