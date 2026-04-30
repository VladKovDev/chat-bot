package action

import (
	"context"

	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
)

// Action represents a business operation that can be executed during a transition
type Action interface {
	Execute(ctx context.Context, data ActionData) error
}

// ActionData contains information needed to execute an action
type ActionData struct {
	Conversation *conversation.Conversation
	UserText     string
	Context      map[string]interface{}
}