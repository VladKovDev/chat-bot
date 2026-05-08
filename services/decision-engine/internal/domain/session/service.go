package session

import (
	"context"

	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/google/uuid"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// LoadSession loads or creates a session
// For backward compatibility, creates a default user if none exists
func (s *Service) LoadSession(ctx context.Context, chatID int64) (*Session, error) {
	session, err := s.repo.GetByChatID(ctx, chatID)
	if err != nil {
		if err == ErrNotFound {
			// Create new session with a default user ID
			// In production, you would get the user ID from authentication
			defaultUserID := uuid.New() // TODO: Get from authentication context
			session := Session{
				ID:       uuid.New(),
				ChatID:   chatID,
				UserID:   defaultUserID,
				State:    state.StateNew,
				Status:   StatusActive,
				Metadata: make(map[string]interface{}),
			}
			createdSession, err := s.repo.Create(ctx, session)
			if err != nil {
				return nil, err
			}
			return &createdSession, nil
		}
		return nil, err
	}
	return &session, nil
}

// LoadOrCreateSession loads or creates a session with explicit user ID
func (s *Service) LoadOrCreateSession(ctx context.Context, chatID int64, userID uuid.UUID) (*Session, error) {
	session, err := s.repo.GetByChatID(ctx, chatID)
	if err != nil {
		if err == ErrNotFound {
			session := Session{
				ID:       uuid.New(),
				ChatID:   chatID,
				UserID:   userID,
				State:    state.StateNew,
				Status:   StatusActive,
				Metadata: make(map[string]interface{}),
			}
			createdSession, err := s.repo.Create(ctx, session)
			if err != nil {
				return nil, err
			}
			return &createdSession, nil
		}
		return nil, err
	}
	return &session, nil
}

func (s *Service) UpdateSessionState(ctx context.Context, session *Session) (Session, error) {
	return s.repo.UpdateState(ctx, session.ID, session.State)
}

func (s *Service) CloseSession(ctx context.Context, sessionID uuid.UUID) error {
	_, err := s.repo.UpdateStatus(ctx, sessionID, StatusClosed)
	return err
}

// Backward compatibility aliases
func (s *Service) LoadConversation(ctx context.Context, chatID int64) (*Session, error) {
	return s.LoadSession(ctx, chatID)
}

func (s *Service) UpdateConversationState(ctx context.Context, session *Session) (Session, error) {
	return s.UpdateSessionState(ctx, session)
}

func (s *Service) CloseConversation(ctx context.Context, sessionID uuid.UUID) error {
	return s.CloseSession(ctx, sessionID)
}