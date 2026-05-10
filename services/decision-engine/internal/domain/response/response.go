package response

import (
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/google/uuid"
)

// Response represents the decision engine's response to a message
type Response struct {
	Text           string
	Options        []string
	State          state.State
	SessionID      uuid.UUID
	UserMessageID  uuid.UUID
	BotMessageID   uuid.UUID
	Channel        string
	ExternalUserID string
	ClientID       string
	ActiveTopic    string
	Mode           session.Mode
	OperatorStatus session.OperatorStatus
}
