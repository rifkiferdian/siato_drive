package publicdrive

import (
	"github.com/google/uuid"
	"time"
)

type Organization struct {
	ID         uuid.UUID
	Slug, Name string
}
type Node struct {
	ID               uuid.UUID
	Name, Type, MIME string
	Size             int64
	UpdatedAt        time.Time
}
type File struct {
	Node
	StorageKey string
}
