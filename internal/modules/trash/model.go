package trash

import (
	"github.com/google/uuid"
	"time"
)

type Item struct {
	ID                                     uuid.UUID
	Name, Type, MIME, DriveName, DriveType string
	Size                                   int64
	DeletedAt                              time.Time
}
type DeleteResult struct{ StorageKeys []string }
