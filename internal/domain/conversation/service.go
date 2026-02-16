package conversation

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) LoadConversation(ctx context.Context, channel Channel, chatID int64) (*Conversation, error) {
	conv, err := s.repo.GetByChannelAndChatID(ctx, channel, chatID)
	if err != nil {
		if err == ErrNotFound {
			return &Conversation{
				ID:      uuid.New(),
				Channel: channel,
				ChatID:  chatID,
				State:   StateNew,
			}, nil
		}
		return nil, err
	}
	return &conv, nil
}
