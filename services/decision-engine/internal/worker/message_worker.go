package worker

import (
	"context"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/contracts"
	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

type MessageWorker struct {
	convService *conversation.Service
	logger      logger.Logger
	classifier  EventClassifier
}

type EventClassifier interface {
	Classify(ctx context.Context, text string) (state.Event, error)
}

func NewMessageWorker(convService *conversation.Service, logger logger.Logger, classifier EventClassifier) *MessageWorker {
	return &MessageWorker{
		convService: convService,
		logger:      logger,
		classifier:  classifier,
	}
}

func (w *MessageWorker) HandleMessage(ctx context.Context, msg contracts.IncomingMessage) (response.Response, error) {
	// load conversation
	conv, err := w.convService.LoadConversation(ctx, msg.ChatID)
	if err != nil {
		return response.Response{}, fmt.Errorf("failed to load conversation: %w", err)
	}

	// classify event
	event, err := w.classifier.Classify(ctx, msg.Text)
	if err != nil {
		return response.Response{}, fmt.Errorf("failed to classify event: %w", err)
	}

	// check for global events first (these override normal transitions)
	transCtx := conversation.TransitionContext{
		UserText: msg.Text,
	}

	globalResult := conversation.CheckGlobalEvents(event, conv.State, transCtx)

	var newState state.State
	var resp response.Response

	if globalResult.Handled {
		// Global event was triggered, use its result
		newState = globalResult.NewState
		resp = globalResult.Response

		w.logger.Info("Global event triggered",
			w.logger.String("chat_id", fmt.Sprint(conv.ChatID)),
			w.logger.String("event", string(event)),
			w.logger.String("from_state", string(conv.State)),
			w.logger.String("to_state", string(newState)))
	} else {
		// No global event, proceed with normal transition
		newState, resp, err = conv.Transition(event, transCtx)
		if err != nil {
			return response.Response{}, fmt.Errorf("failed to transition conversation: %w", err)
		}
	}

	// update conversation state
	conv.State = newState

	_, err = w.convService.UpdateConversationState(ctx, conv)
	if err != nil {
		return response.Response{}, fmt.Errorf("failed to update conversation state: %w", err)
	}

	w.logger.Info("Response generated",
		w.logger.String("chat_id", fmt.Sprint(conv.ChatID)),
		w.logger.String("text", resp.Text),
		w.logger.String("state", string(newState)))

	return resp, nil
}
