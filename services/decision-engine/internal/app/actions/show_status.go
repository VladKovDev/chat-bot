package actions

import (
	"context"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
)

// ShowStatus retrieves and displays status of an object (booking, payment, etc.)
type ShowStatus struct{}

// Execute retrieves and shows status
func (a *ShowStatus) Execute(ctx context.Context, data action.ActionData) error {
	// TODO: Implement status retrieval logic
	// This should:
	// 1. Extract identifier from context (provided by previous RequestIdentifier action)
	// 2. Query appropriate service to get status
	// 3. Format status information for display
	// 4. Store formatted status in context for response

	fmt.Println("ShowStatus: Retrieving and displaying status")

	// Example implementation:
	// - Get booking/payment ID from context
	// - Call repository to get current status
	// - Format status with relevant details (date, time, status, etc.)
	// - Store in context for presenter

	return nil
}

// NewShowStatus creates a new ShowStatus action
func NewShowStatus() *ShowStatus {
	return &ShowStatus{}
}