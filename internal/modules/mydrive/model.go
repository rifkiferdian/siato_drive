package mydrive

import (
	"github.com/google/uuid"
	"time"
)

type Drive struct {
	ID   uuid.UUID
	Name string
}
type Node struct {
	ID               uuid.UUID
	ParentID         *uuid.UUID
	Name, Type, MIME string
	Size             int64
	UpdatedAt        time.Time
}
type Crumb struct {
	ID   *uuid.UUID
	Name string
}
type Page struct {
	Drive       Drive
	Parent      *Node
	Nodes       []Node
	Breadcrumbs []Crumb
}
type Download struct {
	Node       Node
	StorageKey string
}
