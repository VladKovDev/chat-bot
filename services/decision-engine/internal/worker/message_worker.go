package worker

import (
	"context"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/contracts"
	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

type MessageWorker struct {
	convService *conversation.Service
	logger      logger.Logger
	classifier  EventClassifier
}

type EventClassifier interface {
	Classify(ctx context.Context, text string) (conversation.Event, error)
}

func NewMessageWorker(convService *conversation.Service, logger logger.Logger, classifier EventClassifier) *MessageWorker {
	return &MessageWorker{
		convService: convService,
		logger:      logger,
		classifier:  classifier,
	}
}

func (w *MessageWorker) HandleMessage(ctx context.Context, msg contracts.IncomingMessage) (conversation.BotResponse, error) {
	// load conversation
	conv, err := w.convService.LoadConversation(ctx, msg.Channel, msg.ChatID)
	if err != nil {
		return conversation.BotResponse{}, fmt.Errorf("failed to load conversation: %w", err)
	}

	// classify event
	event, err := w.classifier.Classify(ctx, msg.Text)
	if err != nil {
		return conversation.BotResponse{}, fmt.Errorf("failed to classify event: %w", err)
	}

	// transition conversation
	transCtx := conversation.TransitionContext{
		UserText: msg.Text,
	}

	newState, response, err := conv.Transition(event, transCtx)
	if err != nil {
		return conversation.BotResponse{}, fmt.Errorf("failed to transition conversation: %w", err)
	}

	// update conversation state
	conv.State = newState

	_, err = w.convService.UpdateConversationState(ctx, conv)
	if err != nil {
		return conversation.BotResponse{}, fmt.Errorf("failed to update conversation state: %w", err)
	}

	w.logger.Info("Response generated",
		w.logger.String("chat_id", fmt.Sprint(conv.ChatID)),
		w.logger.String("text", response.Text),
		w.logger.String("state", string(newState)),
		w.logger.String("channel", string(msg.Channel)))

	return response, nil
}
