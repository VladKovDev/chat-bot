package action

import (
	"context"
	"github.com/google/uuid"
)

// LogRepository defines the interface for action log data access
type LogRepository interface {
	// Log creates a new action log entry
	Log(ctx context.Context, entry Log) (Log, error)

	// GetBySessionID retrieves action logs for a session with pagination
	GetBySessionID(ctx context.Context, sessionID uuid.UUID, limit int32, offset int32) ([]Log, error)

	// GetByType retrieves action logs by action type with pagination
	GetByType(ctx context.Context, actionType string, limit int32, offset int32) ([]Log, error)

	// CountBySessionID returns the count of actions for a session
	CountBySessionID(ctx context.Context, sessionID uuid.UUID) (int64, error)
}
