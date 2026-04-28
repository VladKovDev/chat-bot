package contracts

import (
	"time"

	"github.com/google/uuid"
)

type IncomingMessage struct {
	EventID   uuid.UUID
	ChatID    int64
	Text      string
	Timestamp time.Time
}
