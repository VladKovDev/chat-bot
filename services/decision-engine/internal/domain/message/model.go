package message

import (
	"github.com/google/uuid"
	"time"
)

type Message struct {
	ID         uuid.UUID
	SessionID  uuid.UUID
	SenderType SenderType
	Text       string
	Intent     *string
	CreatedAt  time.Time
}

type SenderType string

const (
	SenderTypeUser     SenderType = "user"
	SenderTypeBot      SenderType = "bot"
	SenderTypeOperator SenderType = "operator"
)
