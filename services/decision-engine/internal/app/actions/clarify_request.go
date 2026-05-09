package actions

import (
	"context"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
)

// ClarifyRequest asks user for more specific information
type ClarifyRequest struct{}

// Execute requests clarification from the user
func (a *ClarifyRequest) Execute(ctx context.Context, data action.ActionData) error {
	// TODO: Implement clarification request logic
	// This should:
	// 1. Analyze what information is missing or ambiguous
	// 2. Generate specific questions to clarify user's intent
	// 3. Store clarification questions in context for response
	// 4. Set state to waiting_for_clarification

	fmt.Println("ClarifyRequest: Requesting clarification from user")

	// Example implementation:
	// - Determine what aspect needs clarification (category, specific question, etc.)
	// - Generate appropriate clarification questions
	// - Store in context for presenter to format

	return nil
}

// NewClarifyRequest creates a new ClarifyRequest action
func NewClarifyRequest() *ClarifyRequest {
	return &ClarifyRequest{}
}