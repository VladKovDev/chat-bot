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

	// GetByChannelAndChatID retrieves a conversation by channel and chat ID
	GetByChannelAndChatID(ctx context.Context, channel Channel, chatID int64) (Conversation, error)

	// UpdateState updates the conversation state
	UpdateState(ctx context.Context, id uuid.UUID, state State) (Conversation, error)

	// UpdateStateWithVersion updates the conversation state and increments version
	UpdateStateWithVersion(ctx context.Context, id uuid.UUID, state State) (Conversation, error)

	// ListByChannel retrieves conversations by channel with pagination
	ListByChannel(ctx context.Context, channel Channel, limit int32, offset int32) ([]Conversation, error)

	// ListByState retrieves conversations by state with pagination
	ListByState(ctx context.Context, state State, limit int32, offset int32) ([]Conversation, error)

	// Delete deletes a conversation by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// CountByChannel returns the count of conversations by channel
	CountByChannel(ctx context.Context, channel Channel) (int64, error)
}