package actions

import (
	"context"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
)

// RequestIdentifier requests user to provide identifier (booking number, phone, etc.)
type RequestIdentifier struct{}

// Execute requests identifier from user
func (a *RequestIdentifier) Execute(ctx context.Context, data action.ActionData) error {
	// TODO: Implement identifier request logic
	// This should:
	// 1. Determine what type of identifier is needed (booking number, phone, email)
	// 2. Generate appropriate request message
	// 3. Store request in context for response
	// 4. Set state to waiting_for_identifier

	fmt.Println("RequestIdentifier: Requesting identifier from user")

	// Example implementation:
	// - Based on intent, determine what identifier is needed
	// - For booking questions: request booking number or phone
	// - For payment questions: request payment ID or order number
	// - Store identifier type in context for validation on next input

	return nil
}

// NewRequestIdentifier creates a new RequestIdentifier action
func NewRequestIdentifier() *RequestIdentifier {
	return &RequestIdentifier{}
}