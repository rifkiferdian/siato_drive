package shareddrive

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }
func (r *Repository) List(ctx context.Context, driveID uuid.UUID, parent *uuid.UUID, q, sort string) ([]Node, error) {
	order := "n.type DESC,LOWER(n.name)"
	if sort == "date" {
		order = "n.updated_at DESC"
	} else if sort == "size" {
		order = "n.size_bytes DESC"
	}
	var p any
	if parent != nil {
		p = parent.String()
	}
	rows, err := r.db.QueryContext(ctx, `SELECT n.id,n.parent_id,n.name,n.type,COALESCE(n.mime_type,''),n.size_bytes,n.updated_at FROM drive_nodes n WHERE n.drive_id=? AND n.parent_id <=> ? AND n.deleted_at IS NULL AND n.status='ready' AND (?='' OR n.name LIKE CONCAT('%',?,'%')) ORDER BY `+order, driveID.String(), p, q, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Node{}
	for rows.Next() {
		var n Node
		var parentID sql.NullString
		if err = rows.Scan(&n.ID, &parentID, &n.Name, &n.Type, &n.MIME, &n.Size, &n.UpdatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			id, _ := uuid.Parse(parentID.String)
			n.ParentID = &id
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
