package actions

import (
	"context"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

// ResetConversationAction resets conversation to initial state
type ResetConversationAction struct {
	repo   conversation.Repository
	logger logger.Logger
}

// NewResetConversationAction creates a new reset conversation action
func NewResetConversationAction(repo conversation.Repository, logger logger.Logger) *ResetConversationAction {
	return &ResetConversationAction{
		repo:   repo,
		logger: logger,
	}
}

// Execute resets the conversation to StateNew and clears metadata
func (a *ResetConversationAction) Execute(ctx context.Context, data action.ActionData) error {
	a.logger.Info("resetting conversation",
		a.logger.String("chat_id", fmt.Sprint(data.Conversation.ChatID)),
		a.logger.String("from_state", string(data.Conversation.State)))

	// Reset state to New
	data.Conversation.State = conversation.StateNew

	// Clear metadata
	data.Conversation.Metadata = make(map[string]interface{})

	// Update in repository
	_, err := a.repo.Update(ctx, *data.Conversation)
	if err != nil {
		a.logger.Error("failed to reset conversation in repository",
			a.logger.String("chat_id", fmt.Sprint(data.Conversation.ChatID)),
			a.logger.Err(err))
		return err
	}

	return nil
}