package usermanagement

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	DisplayName  string
	Role         string
	Active       bool
	StorageBytes int64
	CreatedAt    time.Time
}

func (u User) StorageGB() string {
	return fmt.Sprintf("%.3f GB", float64(u.StorageBytes)/(1024*1024*1024))
}

type CreateInput struct {
	Email, DisplayName, Password, Role string
}

var validRoles = map[string]bool{"admin": true, "user": true}
