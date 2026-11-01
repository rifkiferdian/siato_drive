package auth

import (
	"github.com/google/uuid"
	"time"
)

type User struct {
	ID                 uuid.UUID
	Email, DisplayName string
	Active             bool
}
type Session struct {
	ID, UserID  uuid.UUID
	CSRFToken   string
	ExpiresAt   time.Time
	User        User
	Permissions map[string]bool
}
