package contracts

import (
	"time"

	"github.com/google/uuid"
)

type IncomingMessage struct {
	EventID        uuid.UUID
	SessionID      uuid.UUID
	ChatID         int64
	Channel        string
	ExternalUserID string
	ClientID       string
	Text           string
	Timestamp      time.Time
}
