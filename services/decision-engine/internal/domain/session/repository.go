package session

import (
	"context"

	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/google/uuid"
)

// Repository defines the interface for session data access
type Repository interface {
	// Create creates a new session
	Create(ctx context.Context, session Session) (Session, error)

	// GetByID retrieves a session by its ID
	GetByID(ctx context.Context, id uuid.UUID) (Session, error)

	// GetByChatID retrieves a session by chat ID
	GetByChatID(ctx context.Context, chatID int64) (Session, error)

	// GetByUserID retrieves sessions by user ID
	GetByUserID(ctx context.Context, userID uuid.UUID, limit int32, offset int32) ([]Session, error)

	// Update updates the entire session (state, metadata, etc.)
	Update(ctx context.Context, session Session) (Session, error)

	// UpdateState updates the session state
	UpdateState(ctx context.Context, id uuid.UUID, st state.State) (Session, error)

	// UpdateStateWithVersion updates the session state and increments version
	UpdateStateWithVersion(ctx context.Context, id uuid.UUID, st state.State) (Session, error)

	// UpdateStatus updates the session status
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status) (Session, error)

	// UpdateSummary updates the session summary
	UpdateSummary(ctx context.Context, id uuid.UUID, summary string) (Session, error)

	// List retrieves sessions with pagination
	List(ctx context.Context, limit int32, offset int32) ([]Session, error)

	// ListByState retrieves sessions by state with pagination
	ListByState(ctx context.Context, st state.State, limit int32, offset int32) ([]Session, error)

	// ListByStatus retrieves sessions by status with pagination
	ListByStatus(ctx context.Context, status Status, limit int32, offset int32) ([]Session, error)

	// Delete deletes a session by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// Count returns the count of all sessions
	Count(ctx context.Context) (int64, error)
}
