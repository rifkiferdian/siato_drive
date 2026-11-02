package publicdrive

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }
func (r *Repository) Organization(ctx context.Context, slug string) (Organization, error) {
	var o Organization
	err := r.db.QueryRowContext(ctx, `SELECT id,slug,name FROM org_organizations WHERE slug=? AND active=1`, slug).Scan(&o.ID, &o.Slug, &o.Name)
	return o, err
}
func (r *Repository) PublicNode(ctx context.Context, orgID, nodeID uuid.UUID) (Node, error) {
	var n Node
	var parent sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT n.id,n.parent_id,n.name,n.type,COALESCE(n.mime_type,''),n.size_bytes,n.updated_at FROM drive_nodes n JOIN drive_drives d ON d.id=n.drive_id WHERE n.id=? AND d.organization_id=? AND d.type='public' AND n.deleted_at IS NULL AND n.status='ready'`, nodeID.String(), orgID.String()).Scan(&n.ID, &parent, &n.Name, &n.Type, &n.MIME, &n.Size, &n.UpdatedAt)
	if parent.Valid {
		id, parseErr := uuid.Parse(parent.String)
		if parseErr != nil {
			return n, parseErr
		}
		n.ParentID = &id
	}
	return n, err
}
func (r *Repository) List(ctx context.Context, orgID uuid.UUID, parent *uuid.UUID, q, kind, sort string) ([]Node, error) {
	order := "n.type DESC, lower(n.name) ASC"
	if sort == "date" {
		order = "n.type DESC, n.updated_at DESC"
	} else if sort == "size" {
		order = "n.type DESC, n.size_bytes DESC, lower(n.name) ASC"
	} else if sort == "type" {
		order = "n.type DESC, lower(coalesce(n.mime_type,'')) ASC, lower(n.name) ASC"
	}
	query := `WITH RECURSIVE visible_nodes AS (
		SELECT n.id,n.drive_id,n.parent_id,n.name,n.type,n.mime_type,n.size_bytes,n.updated_at
		FROM drive_nodes n
		JOIN drive_drives d ON d.id=n.drive_id
		WHERE d.organization_id=? AND d.type='public' AND n.parent_id IS NULL AND n.deleted_at IS NULL AND n.status='ready'
		UNION ALL
		SELECT child.id,child.drive_id,child.parent_id,child.name,child.type,child.mime_type,child.size_bytes,child.updated_at
		FROM drive_nodes child
		JOIN visible_nodes parent ON parent.id=child.parent_id AND parent.drive_id=child.drive_id
		WHERE child.deleted_at IS NULL AND child.status='ready'
	)
	SELECT n.id,n.parent_id,n.name,n.type,coalesce(n.mime_type,''),n.size_bytes,n.updated_at
	FROM visible_nodes n
	WHERE ((?='' AND n.parent_id <=> ?) OR (?<>'' AND n.name LIKE CONCAT('%',?,'%')))
	AND (?='' OR n.type=? OR n.mime_type LIKE CONCAT(?,'%'))
	ORDER BY ` + order
	var pid any
	if parent != nil {
		pid = parent.String()
	}
	rows, err := r.db.QueryContext(ctx, query, orgID.String(), q, pid, q, q, kind, kind, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Node{}
	for rows.Next() {
		var n Node
		var parent sql.NullString
		if err = rows.Scan(&n.ID, &parent, &n.Name, &n.Type, &n.MIME, &n.Size, &n.UpdatedAt); err != nil {
			return nil, err
		}
		if parent.Valid {
			id, _ := uuid.Parse(parent.String)
			n.ParentID = &id
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
func (r *Repository) PublicFile(ctx context.Context, orgID, nodeID uuid.UUID) (File, error) {
	var f File
	err := r.db.QueryRowContext(ctx, `SELECT n.id,n.name,n.type,coalesce(n.mime_type,''),n.size_bytes,n.updated_at,o.storage_key FROM drive_nodes n JOIN drive_drives d ON d.id=n.drive_id JOIN storage_objects o ON o.id=n.storage_object_id WHERE n.id=? AND d.organization_id=? AND d.type='public' AND n.type='file' AND n.deleted_at IS NULL AND n.status='ready' AND o.state='ready'`, nodeID.String(), orgID.String()).Scan(&f.ID, &f.Name, &f.Type, &f.MIME, &f.Size, &f.UpdatedAt, &f.StorageKey)
	return f, err
}
func (r *Repository) LogDownload(ctx context.Context, orgID, nodeID uuid.UUID, ip string) {
	_, _ = r.db.ExecContext(ctx, `INSERT INTO audit_download_logs(organization_id,node_id,ip) VALUES(?,?,NULLIF(?,''))`, orgID.String(), nodeID.String(), ip)
}
