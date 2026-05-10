package response

import (
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/google/uuid"
)

type QuickReply struct {
	ID      string
	Label   string
	Action  string
	Payload map[string]any
}

// Response represents the decision engine's response to a message
type Response struct {
	Text           string
	Options        []string
	QuickReplies   []QuickReply
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
