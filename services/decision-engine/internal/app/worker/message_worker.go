package worker

import (
	"context"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/app/presenter"
	"github.com/VladKovDev/chat-bot/internal/app/processor"
	"github.com/VladKovDev/chat-bot/internal/app/transition"
	"github.com/VladKovDev/chat-bot/internal/contracts"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/nlp"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

type MessageWorker struct {
	convService    *conversation.Service
	transitionEng  *transition.Engine
	processor      *processor.Processor
	presenter      *presenter.Presenter
	nlpClassifier  *nlp.Classifier
	logger         logger.Logger
}

func NewMessageWorker(
	convService *conversation.Service,
	transitionEng *transition.Engine,
	proc *processor.Processor,
	pr *presenter.Presenter,
	nlpClassifier *nlp.Classifier,
	logger logger.Logger,
) *MessageWorker {
	return &MessageWorker{
		convService:   convService,
		transitionEng: transitionEng,
		processor:     proc,
		presenter:     pr,
		nlpClassifier: nlpClassifier,
		logger:        logger,
	}
}

func (w *MessageWorker) HandleMessage(ctx context.Context, msg contracts.IncomingMessage) (response.Response, error) {
	// 1. Load conversation
	conv, err := w.convService.LoadConversation(ctx, msg.ChatID)
	if err != nil {
		return response.Response{}, fmt.Errorf("failed to load conversation: %w", err)
	}

	w.logger.Debug("conversation loaded",
		w.logger.String("chat_id", fmt.Sprint(conv.ChatID)),
		w.logger.String("state", string(conv.State)))

	// 2. Classify event (Intent → Event adapter inside)
	event, err := w.nlpClassifier.Classify(ctx, msg.Text)
	if err != nil {
		return response.Response{}, fmt.Errorf("failed to classify: %w", err)
	}

	w.logger.Debug("event classified",
		w.logger.String("chat_id", fmt.Sprint(conv.ChatID)),
		w.logger.String("event", string(event)))

	// 3. Execute transition
	result, err := w.transitionEng.Execute(ctx, conv.State, event)
	if err != nil {
		w.logger.Error("transition failed, staying in current state",
			w.logger.String("chat_id", fmt.Sprint(conv.ChatID)),
			w.logger.Err(err))
		// Continue anyway with fallback response
		result = &transition.TransitionResult{
			NextState:   conv.State,
			Actions:     []string{},
			ResponseKey: "error",
		}
	}

	w.logger.Debug("transition executed",
		w.logger.String("chat_id", fmt.Sprint(conv.ChatID)),
		w.logger.String("from", string(conv.State)),
		w.logger.String("to", string(result.NextState)),
		w.logger.Int("actions", len(result.Actions)))

	// 4. Execute actions
	if len(result.Actions) > 0 {
		actionData := action.ActionData{
			Conversation: conv,
			UserText:     msg.Text,
			Context:      make(map[string]interface{}),
		}

		if err := w.processor.Execute(ctx, result.Actions, actionData); err != nil {
			w.logger.Error("actions execution failed",
				w.logger.String("chat_id", fmt.Sprint(conv.ChatID)),
				w.logger.Err(err))
			// Continue anyway to return response
		}
	}

	// 5. Update conversation state
	conv.State = result.NextState
	if _, err := w.convService.UpdateConversationState(ctx, conv); err != nil {
		return response.Response{}, fmt.Errorf("failed to update state: %w", err)
	}

	w.logger.Debug("state updated",
		w.logger.String("chat_id", fmt.Sprint(conv.ChatID)),
		w.logger.String("state", string(conv.State)))

	// 6. Present response
	resp, err := w.presenter.Present(result.ResponseKey, conv.State)
	if err != nil {
		w.logger.Error("failed to present response",
			w.logger.String("chat_id", fmt.Sprint(conv.ChatID)),
			w.logger.Err(err))
		return response.Response{}, fmt.Errorf("failed to present response: %w", err)
	}

	w.logger.Info("response generated",
		w.logger.String("chat_id", fmt.Sprint(conv.ChatID)),
		w.logger.String("text", resp.Text),
		w.logger.String("state", fmt.Sprint(resp.State)),
		w.logger.Int("options_count", len(resp.Options)))

	return resp, nil
}