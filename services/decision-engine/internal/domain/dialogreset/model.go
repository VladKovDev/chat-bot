package dialogreset

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Request struct {
	SessionID uuid.UUID
	Actor     string
	Reason    string
}

type Summary struct {
	SessionID uuid.UUID
	Existed   bool
	Deleted   map[string]int64
	AuditID   uuid.UUID
	CreatedAt time.Time
}

type Repository interface {
	ResetSession(ctx context.Context, req Request) (Summary, error)
}
