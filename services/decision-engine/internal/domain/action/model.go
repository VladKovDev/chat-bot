package action

import (
	"github.com/google/uuid"
	"time"
)

// Log represents an action execution record for audit trail
type Log struct {
	ID              uuid.UUID
	SessionID       uuid.UUID
	ActionType      string
	RequestPayload  map[string]interface{}
	ResponsePayload map[string]interface{}
	Error           *string
	CreatedAt       time.Time
}
