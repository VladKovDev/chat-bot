package actions

import (
	"context"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
)

// ReturnToMenu returns conversation to main menu
type ReturnToMenu struct{}

// Execute resets conversation state to main menu
func (a *ReturnToMenu) Execute(ctx context.Context, data action.ActionData) error {
	// TODO: Implement return to menu logic
	// This should:
	// 1. Clear temporary context data (keep session metadata)
	// 2. Reset state to waiting_for_category or new
	// 3. Store flag in context to show main menu response

	fmt.Println("ReturnToMenu: Returning to main menu")

	// Example implementation:
	// - Clear conversation-specific context
	// - Reset state to StateWaitingForCategory or StateNew
	// - Set flag to show main menu options
	// - Keep user session data intact

	if data.Session != nil {
		data.Session.State = state.StateWaitingForCategory
	}

	return nil
}

// NewReturnToMenu creates a new ReturnToMenu action
func NewReturnToMenu() *ReturnToMenu {
	return &ReturnToMenu{}
}
