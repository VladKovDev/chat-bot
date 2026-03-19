package contracts

import (
	"time"

	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
	"github.com/google/uuid"
)

type IncomingMessage struct {
	EventID   uuid.UUID
	Channel   conversation.Channel
	ChatID    int64
	Text      string
	Timestamp time.Time
}
