package trash

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"sort"
)

var ErrNotFound = errors.New("resource not found")
var ErrParentDeleted = errors.New("parent folder masih berada di Sampah")

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }
func accessSQL() string {
	return `(d.type='personal' AND d.owner_user_id=?) OR (d.type='shared' AND EXISTS(SELECT 1 FROM drive_drive_members dm JOIN org_memberships m ON m.user_id=dm.user_id AND m.organization_id=d.organization_id AND m.status='active' WHERE dm.drive_id=d.id AND dm.user_id=? AND dm.role IN ('editor','manager','owner'))) OR (d.type='public' AND ?=1 AND EXISTS(SELECT 1 FROM org_memberships m WHERE m.organization_id=d.organization_id AND m.user_id=? AND m.status='active'))`
}
func (r *Repository) List(ctx context.Context, userID uuid.UUID, includePublic bool, q, kind string) ([]Item, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT n.id,n.name,n.type,COALESCE(n.mime_type,''),n.size_bytes,n.deleted_at,d.name,d.type FROM drive_nodes n JOIN drive_drives d ON d.id=n.drive_id WHERE n.deleted_at IS NOT NULL AND (`+accessSQL()+`) AND (?='' OR n.name LIKE CONCAT('%',?,'%')) AND (?='' OR d.type=?) ORDER BY n.deleted_at DESC`, userID.String(), userID.String(), includePublic, userID.String(), q, q, kind, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Item{}
	for rows.Next() {
		var item Item
		if err = rows.Scan(&item.ID, &item.Name, &item.Type, &item.MIME, &item.Size, &item.DeletedAt, &item.DriveName, &item.DriveType); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
func (r *Repository) accessibleNode(ctx context.Context, tx *sql.Tx, userID, nodeID uuid.UUID, includePublic bool) (driveID uuid.UUID, parent *uuid.UUID, err error) {
	var parentRaw sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT n.drive_id,n.parent_id FROM drive_nodes n JOIN drive_drives d ON d.id=n.drive_id WHERE n.id=? AND n.deleted_at IS NOT NULL AND (`+accessSQL()+`)`, nodeID.String(), userID.String(), userID.String(), includePublic, userID.String()).Scan(&driveID, &parentRaw)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	if parentRaw.Valid {
		id, _ := uuid.Parse(parentRaw.String)
		parent = &id
	}
	return
}
func (r *Repository) Restore(ctx context.Context, userID, nodeID uuid.UUID, includePublic bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	driveID, parent, err := r.accessibleNode(ctx, tx, userID, nodeID, includePublic)
	if err != nil {
		return err
	}
	if parent != nil {
		var deleted sql.NullTime
		if err = tx.QueryRowContext(ctx, `SELECT deleted_at FROM drive_nodes WHERE id=? AND drive_id=?`, parent.String(), driveID.String()).Scan(&deleted); err != nil {
			return ErrParentDeleted
		}
		if deleted.Valid {
			return ErrParentDeleted
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE drive_nodes SET deleted_at=NULL,updated_by=?,updated_at=UTC_TIMESTAMP(6) WHERE id=? AND drive_id=?`, userID.String(), nodeID.String(), driveID.String()); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) PermanentDelete(ctx context.Context, userID, nodeID uuid.UUID, includePublic bool) (DeleteResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return DeleteResult{}, err
	}
	defer tx.Rollback()
	driveID, _, err := r.accessibleNode(ctx, tx, userID, nodeID, includePublic)
	if err != nil {
		return DeleteResult{}, err
	}
	rows, err := tx.QueryContext(ctx, `WITH RECURSIVE tree AS (SELECT id,parent_id,0 depth,storage_object_id FROM drive_nodes WHERE id=? AND drive_id=? UNION ALL SELECT n.id,n.parent_id,t.depth+1,n.storage_object_id FROM drive_nodes n JOIN tree t ON n.parent_id=t.id WHERE n.drive_id=?) SELECT t.id,t.depth,t.storage_object_id,o.storage_key FROM tree t LEFT JOIN storage_objects o ON o.id=t.storage_object_id ORDER BY t.depth DESC`, nodeID.String(), driveID.String(), driveID.String())
	if err != nil {
		return DeleteResult{}, err
	}
	type row struct {
		id       string
		depth    int
		objectID sql.NullString
		key      sql.NullString
	}
	all := []row{}
	for rows.Next() {
		var x row
		if err = rows.Scan(&x.id, &x.depth, &x.objectID, &x.key); err != nil {
			rows.Close()
			return DeleteResult{}, err
		}
		all = append(all, x)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return DeleteResult{}, err
	}
	rows.Close()
	sort.SliceStable(all, func(i, j int) bool { return all[i].depth > all[j].depth })
	result := DeleteResult{}
	objects := []string{}
	for _, x := range all {
		if _, err = tx.ExecContext(ctx, `DELETE FROM drive_nodes WHERE id=?`, x.id); err != nil {
			return DeleteResult{}, err
		}
		if x.objectID.Valid {
			objects = append(objects, x.objectID.String)
		}
		if x.key.Valid {
			result.StorageKeys = append(result.StorageKeys, x.key.String)
		}
	}
	for _, id := range objects {
		if _, err = tx.ExecContext(ctx, `DELETE FROM storage_objects WHERE id=?`, id); err != nil {
			return DeleteResult{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return DeleteResult{}, err
	}
	return result, nil
}
