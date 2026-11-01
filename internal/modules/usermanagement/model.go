package usermanagement

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID
	Email       string
	DisplayName string
	Role        string
	Active      bool
	CreatedAt   time.Time
}

type CreateInput struct {
	Email, DisplayName, Password, Role string
}

var validRoles = map[string]bool{"admin": true, "user": true}
