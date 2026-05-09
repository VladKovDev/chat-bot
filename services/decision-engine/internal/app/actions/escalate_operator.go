package actions

import (
	"context"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

// EscalateOperatorAction escalates conversation to human operator
type EscalateOperatorAction struct {
	logger logger.Logger
}

// NewEscalateOperatorAction creates a new escalate operator action
func NewEscalateOperatorAction(logger logger.Logger) *EscalateOperatorAction {
	return &EscalateOperatorAction{
		logger: logger,
	}
}

// Execute escalates the conversation to a human operator
// For MVP: just logs the event
// TODO: Create ticket in operator system
func (a *EscalateOperatorAction) Execute(ctx context.Context, data action.ActionData) error {
	a.logger.Info("escalating to operator",
		a.logger.String("chat_id", fmt.Sprint(data.Session.ChatID)),
		a.logger.String("current_state", string(data.Session.State)))

	// TODO: Create ticket in operator system (CRM, helpdesk, etc.)
	return nil
}