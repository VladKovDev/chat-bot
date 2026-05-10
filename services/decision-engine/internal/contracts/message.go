package contracts

import (
	"time"

	"github.com/google/uuid"
)

type IncomingMessage struct {
	EventID        uuid.UUID
	SessionID      uuid.UUID
	Channel        string
	ExternalUserID string
	ClientID       string
	Text           string
	QuickReply     *QuickReplySelection
	RequestID      string
	Timestamp      time.Time
}

type QuickReplySelection struct {
	ID      string
	Label   string
	Action  string
	Payload map[string]any
}
