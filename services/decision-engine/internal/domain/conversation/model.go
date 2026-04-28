package conversation

import (
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/google/uuid"
)

type Conversation struct {
	ID     uuid.UUID
	ChatID int64
	State  state.State
}
