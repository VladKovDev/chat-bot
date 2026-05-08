package user

import (
	"github.com/google/uuid"
	"time"
)

type User struct {
	ID         uuid.UUID
	ExternalID string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
