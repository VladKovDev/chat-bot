package conversation

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for conversation data access
type Repository interface {
	// Create creates a new conversation
	Create(ctx context.Context, conv Conversation) (Conversation, error)

	// GetByID retrieves a conversation by its ID
	GetByID(ctx context.Context, id uuid.UUID) (Conversation, error)

	// GetByChatID retrieves a conversation by chat ID
	GetByChatID(ctx context.Context, chatID int64) (Conversation, error)

	// Update updates the entire conversation (state, metadata, etc.)
	Update(ctx context.Context, conv Conversation) (Conversation, error)

	// UpdateState updates the conversation state
	UpdateState(ctx context.Context, id uuid.UUID, state State) (Conversation, error)

	// UpdateStateWithVersion updates the conversation state and increments version
	UpdateStateWithVersion(ctx context.Context, id uuid.UUID, state State) (Conversation, error)

	// List retrieves conversations with pagination
	List(ctx context.Context, limit int32, offset int32) ([]Conversation, error)

	// ListByState retrieves conversations by state with pagination
	ListByState(ctx context.Context, state State, limit int32, offset int32) ([]Conversation, error)

	// Delete deletes a conversation by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// Count returns the count of all conversations
	Count(ctx context.Context) (int64, error)
}
