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
type ShareLink struct {
	Token         string
	ExpiresAt     *time.Time
	AllowDownload bool
}
type SharedRoot struct {
	Node
	DriveID       uuid.UUID
	OwnerName     string
	AllowDownload bool
}
type ShareStatus struct {
	NodeID    uuid.UUID  `json:"node_id"`
	Status    string     `json:"status"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}
