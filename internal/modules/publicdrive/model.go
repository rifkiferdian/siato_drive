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
	ParentID         *uuid.UUID
	Name, Type, MIME string
	Size             int64
	UpdatedAt        time.Time
}
type Drive struct {
	ID           uuid.UUID
	Name         string
	Organization Organization
}
type Crumb struct {
	ID   *uuid.UUID
	Name string
}
type ManagerPage struct {
	Drive       Drive
	Parent      *Node
	Nodes       []Node
	Breadcrumbs []Crumb
}
type File struct {
	Node
	StorageKey string
}
