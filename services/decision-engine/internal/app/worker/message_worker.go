package worker

import (
	"context"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/app/presenter"
	"github.com/VladKovDev/chat-bot/internal/app/processor"
	"github.com/VladKovDev/chat-bot/internal/contracts"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/message"
	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/llm"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

type MessageWorker struct {
	sessionService *session.Service
	processor      *processor.Processor
	presenter      *presenter.Presenter
	messageRepo    message.Repository
	llmClient      *llm.Client
	logger         logger.Logger
}

func NewMessageWorker(
	sessionService *session.Service,
	proc *processor.Processor,
	pr *presenter.Presenter,
	messageRepo message.Repository,
	llmClient *llm.Client,
	logger logger.Logger,
) *MessageWorker {
	return &MessageWorker{
		sessionService: sessionService,
		processor:      proc,
		presenter:      pr,
		messageRepo:    messageRepo,
		llmClient:      llmClient,
		logger:         logger,
	}
}

func (w *MessageWorker) HandleMessage(ctx context.Context, msg contracts.IncomingMessage) (response.Response, error) {
	// 1. Load session first
	sess, err := w.sessionService.LoadSession(ctx, msg.ChatID)
	if err != nil {
		return response.Response{}, fmt.Errorf("failed to load session: %w", err)
	}

	w.logger.Debug("session loaded",
		w.logger.String("chat_id", fmt.Sprint(sess.ChatID)),
		w.logger.String("state", string(sess.State)))

	// 2. Save incoming message to DB
	incomingMsg := message.Message{
		SessionID:  sess.ID,
		SenderType: message.SenderTypeUser,
		Text:       msg.Text,
		CreatedAt:  msg.Timestamp,
	}
	if _, err := w.messageRepo.Create(ctx, incomingMsg); err != nil {
		w.logger.Warn("failed to save message",
			w.logger.String("chat_id", fmt.Sprint(sess.ChatID)),
			w.logger.Err(err))
		// Continue anyway, message saving is not critical
	}

	// 3. Load last 10 messages (now includes the message we just saved)
	history, err := w.messageRepo.GetLastMessagesBySessionID(ctx, sess.ID, 10)
	if err != nil {
		w.logger.Warn("failed to load message history",
			w.logger.String("chat_id", fmt.Sprint(sess.ChatID)),
			w.logger.Err(err))
		history = []message.Message{}
	}

	// 4. Convert to LLM format
	llmMessages := w.convertToLLMMessages(history)

	// 5. Build summary
	summary := ""
	if sess.Summary != nil {
		summary = *sess.Summary
	}

	// 6. Call LLM /decide
	decideReq := contracts.DecideLLMRequest{
		State:    string(sess.State),
		Summary:  summary,
		Messages: llmMessages,
	}

	decideRespRaw, err := w.llmClient.Decide(ctx, decideReq)
	if err != nil {
		w.logger.Error("LLM decide failed",
			w.logger.String("chat_id", fmt.Sprint(sess.ChatID)),
			w.logger.Err(err))
		return response.Response{}, fmt.Errorf("LLM decide failed: %w", err)
	}

	decideResp, err := w.parseDecideResponse(decideRespRaw)
	if err != nil {
		w.logger.Error("invalid decide response",
			w.logger.String("chat_id", fmt.Sprint(sess.ChatID)),
			w.logger.Err(err))
		return response.Response{}, fmt.Errorf("invalid decide response: %w", err)
	}

	w.logger.Debug("LLM decide response",
		w.logger.String("chat_id", fmt.Sprint(sess.ChatID)),
		w.logger.String("intent", decideResp.Intent),
		w.logger.String("next_state", decideResp.State),
		w.logger.Int("actions", len(decideResp.Actions)))

	// 7. Execute actions and collect results
	actionData := action.ActionData{
		Session:  sess,
		UserText:  msg.Text,
		Context:  make(map[string]interface{}),
	}

	actionResults := w.processor.ExecuteWithResults(ctx, decideResp.Actions, actionData)

	// 8. Select response based on state and action results
	responseKey, err := w.processor.SelectResponse(ctx, state.State(decideResp.State), actionResults)
	if err != nil {
		w.logger.Error("failed to select response",
			w.logger.String("chat_id", fmt.Sprint(sess.ChatID)),
			w.logger.Err(err))
		return response.Response{}, fmt.Errorf("failed to select response: %w", err)
	}

	// 9. Update session state
	sess.State = state.State(decideResp.State)
	if _, err := w.sessionService.UpdateSessionState(ctx, sess); err != nil {
		w.logger.Error("failed to update session state",
			w.logger.String("chat_id", fmt.Sprint(sess.ChatID)),
			w.logger.Err(err))
		return response.Response{}, fmt.Errorf("failed to update state: %w", err)
	}

	// 10. Present response
	resp, err := w.presenter.Present(responseKey, sess.State)
	if err != nil {
		w.logger.Error("failed to present response",
			w.logger.String("chat_id", fmt.Sprint(sess.ChatID)),
			w.logger.Err(err))
		return response.Response{}, fmt.Errorf("failed to present: %w", err)
	}

	w.logger.Info("response generated",
		w.logger.String("chat_id", fmt.Sprint(sess.ChatID)),
		w.logger.String("response_key", responseKey),
		w.logger.String("state", string(resp.State)),
		w.logger.String("text", resp.Text))

	return resp, nil
}

func (w *MessageWorker) convertToLLMMessages(messages []message.Message) []contracts.LLMMessage {
	result := make([]contracts.LLMMessage, len(messages))
	for i, msg := range messages {
		role := string(msg.SenderType)
		result[len(messages)-1-i] = contracts.LLMMessage{ // Reverse to chronological order
			Role: role,
			Text: msg.Text,
		}
	}
	return result
}

func (w *MessageWorker) parseDecideResponse(raw interface{}) (*contracts.DecideLLMResponse, error) {
	data, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected map, got %T", raw)
	}

	intent, _ := data["intent"].(string)
	st, _ := data["state"].(string)

	actionsRaw, _ := data["actions"].([]interface{})
	actions := make([]string, len(actionsRaw))
	for i, a := range actionsRaw {
		actions[i], _ = a.(string)
	}

	return &contracts.DecideLLMResponse{
		Intent:  intent,
		State:   st,
		Actions: actions,
	}, nil
}