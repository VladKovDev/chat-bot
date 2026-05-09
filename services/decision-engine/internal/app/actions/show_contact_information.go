package actions

import (
	"context"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
)

// ShowContactInformation displays contact information
type ShowContactInformation struct{}

// Execute displays contact information
func (a *ShowContactInformation) Execute(ctx context.Context, data action.ActionData) error {
	// TODO: Implement contact information display logic
	// This should:
	// 1. Retrieve contact information from config
	// 2. Format contact details (phone, email, address, hours)
	// 3. Store in context for response

	fmt.Println("ShowContactInformation: Displaying contact information")

	// Example implementation:
	// - Get contact info from configuration
	// - Format with phone, email, website
	// - Include working hours
	// - Store in context for presenter

	return nil
}

// NewShowContactInformation creates a new ShowContactInformation action
func NewShowContactInformation() *ShowContactInformation {
	return &ShowContactInformation{}
}
