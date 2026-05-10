package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/VladKovDev/chat-bot/internal/app/presenter"
	"github.com/VladKovDev/chat-bot/internal/app/processor"
	"github.com/VladKovDev/chat-bot/internal/apperror"
	"github.com/VladKovDev/chat-bot/internal/contracts"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/message"
	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/llm"
	"github.com/VladKovDev/chat-bot/internal/observability"
	"github.com/VladKovDev/chat-bot/pkg/logger"
	"github.com/google/uuid"
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
	sess, err := w.loadSession(ctx, msg)
	if err != nil {
		return response.Response{}, apperror.Wrap(apperror.CodeDatabaseUnavailable, "load_session", err)
	}

	w.logger.Debug("session loaded",
		w.logger.String("request_id", msg.RequestID),
		w.logger.String("session_id", sess.ID.String()),
		w.logger.String("channel", sess.Channel),
		w.logger.String("state", string(sess.State)))

	// 2. Save incoming message to DB
	incomingMsg := message.Message{
		SessionID:  sess.ID,
		SenderType: message.SenderTypeUser,
		Text:       msg.Text,
		CreatedAt:  msg.Timestamp,
	}
	createdMsg, err := w.messageRepo.Create(ctx, incomingMsg)
	if err != nil {
		w.logger.Error("failed to save inbound message",
			w.logger.String("request_id", msg.RequestID),
			w.logger.String("session_id", sess.ID.String()),
			w.logger.String("error_code", string(apperror.CodeDatabaseUnavailable)))
		return response.Response{}, apperror.Wrap(apperror.CodeDatabaseUnavailable, "save_message", err)
	}

	// 3. Load last 10 messages (now includes the message we just saved)
	history, err := w.messageRepo.GetLastMessagesBySessionID(ctx, sess.ID, 10)
	if err != nil {
		w.logger.Error("failed to load message history",
			w.logger.String("request_id", msg.RequestID),
			w.logger.String("session_id", sess.ID.String()),
			w.logger.String("message_id", createdMsg.ID.String()),
			w.logger.String("error_code", string(apperror.CodeDatabaseUnavailable)))
		return response.Response{}, apperror.Wrap(apperror.CodeDatabaseUnavailable, "load_message_history", err)
	}

	// 4. Convert to LLM format
	llmMessages := w.convertToLLMMessages(history)

	// 5. Call LLM /decide
	decideReq := contracts.DecideLLMRequest{
		State:    string(sess.State),
		Summary:  "",
		Messages: llmMessages,
	}

	decideRespRaw, err := w.llmClient.Decide(ctx, decideReq)
	if err != nil {
		w.logger.Error("LLM decide failed",
			w.logger.String("request_id", msg.RequestID),
			w.logger.String("session_id", sess.ID.String()),
			w.logger.String("message_id", createdMsg.ID.String()),
			w.logger.String("error_code", string(apperror.PublicFromError(err, msg.RequestID).Code)))
		return response.Response{}, apperror.Wrap(apperror.CodeProviderUnavailable, "llm_decide", err)
	}

	decideResp, err := w.parseDecideResponse(decideRespRaw)
	if err != nil {
		w.logger.Error("invalid decide response",
			w.logger.String("request_id", msg.RequestID),
			w.logger.String("session_id", sess.ID.String()),
			w.logger.String("message_id", createdMsg.ID.String()),
			w.logger.String("error_code", string(apperror.CodeProviderUnavailable)))
		return response.Response{}, apperror.Wrap(apperror.CodeProviderUnavailable, "parse_llm_decision", err)
	}

	w.logger.Debug("LLM decide response",
		w.logger.String("request_id", msg.RequestID),
		w.logger.String("session_id", sess.ID.String()),
		w.logger.String("message_id", createdMsg.ID.String()),
		w.logger.String("intent", decideResp.Intent),
		w.logger.String("next_state", decideResp.State),
		w.logger.Int("actions", len(decideResp.Actions)))

	// 6. Execute actions and collect results
	actionData := action.ActionData{
		Session:  sess,
		UserText: msg.Text,
		Context:  make(map[string]interface{}),
	}

	actionResults := w.processor.ExecuteWithResults(ctx, decideResp.Actions, actionData)

	// 7. Select response based on state and action results
	responseKey, err := w.processor.SelectResponse(ctx, state.State(decideResp.State), actionResults)
	if err != nil {
		w.logger.Error("failed to select response",
			w.logger.String("request_id", msg.RequestID),
			w.logger.String("session_id", sess.ID.String()),
			w.logger.String("message_id", createdMsg.ID.String()),
			w.logger.String("error_code", string(apperror.CodeProcessingFailed)))
		return response.Response{}, apperror.Wrap(apperror.CodeProcessingFailed, "select_response", err)
	}

	// 8. Update session state
	sess.State = state.State(decideResp.State)
	topic := activeTopicForState(sess.State, sess.ActiveTopic)
	contextDecision := session.ContextDecision{
		Intent:        decideResp.Intent,
		Topic:         topic,
		LowConfidence: isLowConfidence(decideResp.Confidence),
		Event:         eventForDecision(sess.Mode, decideResp),
		Metadata: map[string]interface{}{
			"last_decision_state": decideResp.State,
		},
	}
	if _, err := w.sessionService.ApplyContextDecision(ctx, sess, contextDecision); err != nil {
		w.logger.Error("failed to update session state",
			w.logger.String("request_id", msg.RequestID),
			w.logger.String("session_id", sess.ID.String()),
			w.logger.String("message_id", createdMsg.ID.String()),
			w.logger.String("error_code", string(apperror.CodeDatabaseUnavailable)))
		return response.Response{}, apperror.Wrap(apperror.CodeDatabaseUnavailable, "update_session_context", err)
	}

	// 9. Present response
	resp, err := w.presenter.Present(responseKey, sess.State)
	if err != nil {
		w.logger.Error("failed to present response",
			w.logger.String("request_id", msg.RequestID),
			w.logger.String("session_id", sess.ID.String()),
			w.logger.String("message_id", createdMsg.ID.String()),
			w.logger.String("error_code", string(apperror.CodeProcessingFailed)))
		return response.Response{}, apperror.Wrap(apperror.CodeProcessingFailed, "present_response", err)
	}

	botMsg := message.Message{
		SessionID:  sess.ID,
		SenderType: message.SenderTypeBot,
		Text:       resp.Text,
		CreatedAt:  time.Now().UTC(),
	}
	createdBotMsg, err := w.messageRepo.Create(ctx, botMsg)
	if err != nil {
		w.logger.Error("failed to save bot message",
			w.logger.String("request_id", msg.RequestID),
			w.logger.String("session_id", sess.ID.String()),
			w.logger.String("message_id", createdMsg.ID.String()),
			w.logger.String("error_code", string(apperror.CodeDatabaseUnavailable)))
		return response.Response{}, apperror.Wrap(apperror.CodeDatabaseUnavailable, "save_bot_message", err)
	}

	w.logger.Info("response generated",
		w.logger.String("request_id", msg.RequestID),
		w.logger.String("session_id", sess.ID.String()),
		w.logger.String("message_id", createdBotMsg.ID.String()),
		w.logger.String("response_key", responseKey),
		w.logger.String("state", string(resp.State)),
		w.logger.Int("response_text_length", observability.LenForLog(resp.Text)))

	resp.SessionID = sess.ID
	resp.UserMessageID = createdMsg.ID
	resp.BotMessageID = createdBotMsg.ID
	resp.Channel = sess.Channel
	resp.ExternalUserID = sess.ExternalUserID
	resp.ClientID = sess.ClientID
	resp.ActiveTopic = sess.ActiveTopic
	resp.Mode = sess.Mode
	resp.OperatorStatus = sess.OperatorStatus

	return resp, nil
}

func (w *MessageWorker) StartSession(ctx context.Context, identity session.Identity) (session.StartResult, error) {
	return w.sessionService.StartSession(ctx, identity)
}

func (w *MessageWorker) loadSession(ctx context.Context, msg contracts.IncomingMessage) (*session.Session, error) {
	identity := session.Identity{
		Channel:        msg.Channel,
		ExternalUserID: msg.ExternalUserID,
		ClientID:       msg.ClientID,
	}

	if msg.SessionID != uuid.Nil {
		return w.sessionService.LoadSessionByID(ctx, msg.SessionID, identity)
	}

	if err := session.ValidateIdentity(identity); err == nil {
		result, err := w.sessionService.StartSession(ctx, identity)
		if err != nil {
			return nil, err
		}
		return &result.Session, nil
	}

	if msg.Channel == session.ChannelDevCLI && msg.ChatID != 0 {
		result, err := w.sessionService.StartSession(ctx, session.DevCLIIdentity(msg.ChatID))
		if err != nil {
			return nil, err
		}
		return &result.Session, nil
	}

	return nil, session.ErrInvalidIdentity
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
		Intent:     intent,
		State:      st,
		Actions:    actions,
		Confidence: parseConfidence(data["confidence"]),
	}, nil
}

func activeTopicForState(st state.State, current string) string {
	switch st {
	case state.StateBooking,
		state.StateWorkspace,
		state.StatePayment,
		state.StateTechIssue,
		state.StateAccount,
		state.StateServices,
		state.StateComplaint,
		state.StateOther:
		return string(st)
	default:
		return current
	}
}

func parseConfidence(raw interface{}) *float64 {
	switch value := raw.(type) {
	case float64:
		return &value
	case float32:
		confidence := float64(value)
		return &confidence
	default:
		return nil
	}
}

func isLowConfidence(confidence *float64) bool {
	return confidence != nil && *confidence < 0.6
}

func eventForDecision(currentMode session.Mode, resp *contracts.DecideLLMResponse) session.Event {
	if containsAction(resp.Actions, action.ActionEscalateToOperator) ||
		containsAction(resp.Actions, "escalate_operator") ||
		resp.Intent == "request_operator" ||
		state.State(resp.State) == state.StateEscalatedToOperator {
		return session.EventRequestOperator
	}

	if state.State(resp.State) == state.StateClosed &&
		(currentMode == session.ModeWaitingOperator || currentMode == session.ModeOperatorConnected) {
		return session.EventOperatorClosed
	}

	return session.EventMessageReceived
}

func containsAction(actions []string, target string) bool {
	for _, name := range actions {
		if name == target {
			return true
		}
	}
	return false
}
