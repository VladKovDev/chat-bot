package conversation

import (
	"context"

	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/google/uuid"
)

type Service struct {
	repo            Repository
	responseLoader  *response.ResponseLoader
}

func NewService(repo Repository, responseLoader *response.ResponseLoader) *Service {
	return &Service{
		repo:           repo,
		responseLoader: responseLoader,
	}
}

func (s *Service) LoadConversation(ctx context.Context, chatID int64) (*Conversation, error) {
	conv, err := s.repo.GetByChatID(ctx, chatID)
	if err != nil {
		if err == ErrNotFound {
			conv := Conversation{
				ID:      uuid.New(),
				ChatID:  chatID,
				State:   state.StateNew,
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

// TransitionWithResponse executes state transition and returns response from loader
func (s *Service) TransitionWithResponse(ctx context.Context, conv *Conversation, event state.Event, userText string) (state.State, response.Response, error) {
	handlerCtx := HandlerContext{
		UserText:        userText,
		ResponseLoader:  s.responseLoader,
		Data:            make(map[string]interface{}),
	}

	newState, resp, err := conv.TransitionWithResponse(event, handlerCtx)
	if err != nil {
		return conv.State, response.Response{}, err
	}
	return newState, resp, nil
}

// GetResponseLoader returns the response loader (useful for testing or external access)
func (s *Service) GetResponseLoader() *response.ResponseLoader {
	return s.responseLoader
}
