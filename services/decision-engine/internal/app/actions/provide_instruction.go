package actions

import (
	"context"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
)

// ProvideInstruction provides step-by-step instructions
type ProvideInstruction struct{}

// Execute provides instructions for a specific action
func (a *ProvideInstruction) Execute(ctx context.Context, data action.ActionData) error {
	// TODO: Implement instruction providing logic
	// This should:
	// 1. Determine what action user needs instructions for
	// 2. Retrieve step-by-step instructions from knowledge base
	// 3. Format instructions clearly with numbered steps
	// 4. Store in context for response

	fmt.Println("ProvideInstruction: Providing step-by-step instructions")

	// Example implementation:
	// - Based on intent (e.g., forgot_password, how_to_book)
	// - Retrieve instructions from knowledge base
	// - Format with clear steps (1, 2, 3...)
	// - Include links or references if applicable
	// - Store in context for presenter

	return nil
}

// NewProvideInstruction creates a new ProvideInstruction action
func NewProvideInstruction() *ProvideInstruction {
	return &ProvideInstruction{}
}
