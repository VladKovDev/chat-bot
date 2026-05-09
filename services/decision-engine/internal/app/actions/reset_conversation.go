package actions

import (
	"context"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

// ResetConversationAction resets conversation to initial state
type ResetConversationAction struct {
	repo   session.Repository
	logger logger.Logger
}

// NewResetConversationAction creates a new reset conversation action
func NewResetConversationAction(repo session.Repository, logger logger.Logger) *ResetConversationAction {
	return &ResetConversationAction{
		repo:   repo,
		logger: logger,
	}
}

// Execute resets the session to StateNew and clears metadata
func (a *ResetConversationAction) Execute(ctx context.Context, data action.ActionData) error {
	a.logger.Info("resetting conversation",
		a.logger.String("chat_id", fmt.Sprint(data.Session.ChatID)),
		a.logger.String("from_state", string(data.Session.State)))

	// Reset state to New
	data.Session.State = state.StateNew

	// Clear metadata
	data.Session.Metadata = make(map[string]interface{})

	// Update in repository
	_, err := a.repo.Update(ctx, *data.Session)
	if err != nil {
		a.logger.Error("failed to reset session in repository",
			a.logger.String("chat_id", fmt.Sprint(data.Session.ChatID)),
			a.logger.Err(err))
		return err
	}

	return nil
}