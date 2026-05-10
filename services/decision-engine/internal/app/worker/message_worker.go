package worker

import (
	"context"
	"time"

	appdecision "github.com/VladKovDev/chat-bot/internal/app/decision"
	"github.com/VladKovDev/chat-bot/internal/app/presenter"
	"github.com/VladKovDev/chat-bot/internal/app/processor"
	"github.com/VladKovDev/chat-bot/internal/apperror"
	"github.com/VladKovDev/chat-bot/internal/contracts"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/message"
	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/observability"
	"github.com/VladKovDev/chat-bot/pkg/logger"
	"github.com/google/uuid"
)

type DecisionService interface {
	Decide(ctx context.Context, sess session.Session, history []message.Message, text string) (appdecision.Result, error)
}

type MessageWorker struct {
	sessionService *session.Service
	decision       DecisionService
	processor      *processor.Processor
	presenter      *presenter.Presenter
	messageRepo    message.Repository
	logger         logger.Logger
}

func NewMessageWorker(
	sessionService *session.Service,
	decision DecisionService,
	proc *processor.Processor,
	pr *presenter.Presenter,
	messageRepo message.Repository,
	logger logger.Logger,
) *MessageWorker {
	return &MessageWorker{
		sessionService: sessionService,
		decision:       decision,
		processor:      proc,
		presenter:      pr,
		messageRepo:    messageRepo,
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

	decisionResult, err := w.decision.Decide(ctx, *sess, history, msg.Text)
	if err != nil {
		w.logger.Error("decision service failed",
			w.logger.String("request_id", msg.RequestID),
			w.logger.String("session_id", sess.ID.String()),
			w.logger.String("message_id", createdMsg.ID.String()),
			w.logger.String("error_code", string(apperror.PublicFromError(err, msg.RequestID).Code)))
		return response.Response{}, apperror.Wrap(apperror.CodeProviderUnavailable, "decision_service", err)
	}

	w.logger.Debug("decision resolved",
		w.logger.String("request_id", msg.RequestID),
		w.logger.String("session_id", sess.ID.String()),
		w.logger.String("message_id", createdMsg.ID.String()),
		w.logger.String("intent", decisionResult.Intent),
		w.logger.String("next_state", string(decisionResult.State)),
		w.logger.String("response_key", decisionResult.ResponseKey),
		w.logger.Int("actions", len(decisionResult.Actions)))

	actionData := action.ActionData{
		Session:  sess,
		UserText: msg.Text,
		Context:  make(map[string]interface{}),
	}
	for key, value := range decisionResult.ActionContext {
		actionData.Context[key] = value
	}

	actionResults := w.processor.ExecuteWithResults(ctx, decisionResult.Actions, actionData)

	responseKey := decisionResult.ResponseKey
	if decisionResult.UseActionResponseSelect {
		responseKey, err = w.processor.SelectResponse(ctx, decisionResult.State, actionResults)
		if err != nil {
			w.logger.Error("failed to select response",
				w.logger.String("request_id", msg.RequestID),
				w.logger.String("session_id", sess.ID.String()),
				w.logger.String("message_id", createdMsg.ID.String()),
				w.logger.String("error_code", string(apperror.CodeProcessingFailed)))
			return response.Response{}, apperror.Wrap(apperror.CodeProcessingFailed, "select_response", err)
		}
	}

	sess.State = decisionResult.State
	contextDecision := session.ContextDecision{
		Intent:        decisionResult.Intent,
		Topic:         decisionResult.Topic,
		LowConfidence: decisionResult.LowConfidence,
		Event:         decisionResult.Event,
		Metadata: map[string]interface{}{
			"last_decision_state":   string(decisionResult.State),
			"decision_response_key": responseKey,
		},
	}
	updatedSession, err := w.sessionService.ApplyContextDecision(ctx, sess, contextDecision)
	if err != nil {
		w.logger.Error("failed to update session state",
			w.logger.String("request_id", msg.RequestID),
			w.logger.String("session_id", sess.ID.String()),
			w.logger.String("message_id", createdMsg.ID.String()),
			w.logger.String("error_code", string(apperror.CodeDatabaseUnavailable)))
		return response.Response{}, apperror.Wrap(apperror.CodeDatabaseUnavailable, "update_session_context", err)
	}

	resp, err := w.presenter.Present(responseKey, decisionResult.State)
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
	resp.Channel = updatedSession.Channel
	resp.ExternalUserID = updatedSession.ExternalUserID
	resp.ClientID = updatedSession.ClientID
	resp.ActiveTopic = updatedSession.ActiveTopic
	resp.Mode = updatedSession.Mode
	resp.OperatorStatus = updatedSession.OperatorStatus

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
