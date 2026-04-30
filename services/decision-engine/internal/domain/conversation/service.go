package conversation

import (
	"context"

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

func (s *Service) LoadConversation(ctx context.Context, chatID int64) (*Conversation, error) {
	conv, err := s.repo.GetByChatID(ctx, chatID)
	if err != nil {
		if err == ErrNotFound {
			conv := Conversation{
				ID:       uuid.New(),
				ChatID:   chatID,
				State:    StateNew,
				Metadata: make(map[string]interface{}),
			}
			createdConv, err := s.repo.Create(ctx, conv)
			if err != nil {
				return nil, err
			}
			return &createdConv, nil
		}
		return nil, err
	}
	return &conv, nil
}

func (s *Service) UpdateConversationState(ctx context.Context, conv *Conversation) (Conversation, error) {
	return s.repo.UpdateState(ctx, conv.ID, conv.State)
}