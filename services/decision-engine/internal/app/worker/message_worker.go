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
	operatorDomain "github.com/VladKovDev/chat-bot/internal/domain/operator"
	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/observability"
	"github.com/VladKovDev/chat-bot/pkg/logger"
	"github.com/google/uuid"
)

type DecisionService interface {
	Decide(ctx context.Context, sess session.Session, history []message.Message, text string) (appdecision.Result, error)
}

type QuickReplyDecisionService interface {
	DecideQuickReply(
		ctx context.Context,
		sess session.Session,
		history []message.Message,
		selection appdecision.QuickReplySelection,
		text string,
	) (appdecision.Result, error)
}

type MessageWorker struct {
	sessionService *session.Service
	decision       DecisionService
	processor      *processor.Processor
	presenter      *presenter.Presenter
	persistence    MessagePersistence
	handoff        OperatorHandoffService
	logger         logger.Logger
}

type OperatorHandoffService interface {
	QueueWithDecision(
		ctx context.Context,
		sessionID uuid.UUID,
		reason operatorDomain.Reason,
		snapshot operatorDomain.ContextSnapshot,
		decision session.ContextDecision,
	) (operatorDomain.QueueItem, error)
}

func NewMessageWorker(
	sessionService *session.Service,
	decision DecisionService,
	proc *processor.Processor,
	pr *presenter.Presenter,
	persistence MessagePersistence,
	logger logger.Logger,
	handoff ...OperatorHandoffService,
) *MessageWorker {
	var handoffService OperatorHandoffService
	if len(handoff) > 0 {
		handoffService = handoff[0]
	}
	return &MessageWorker{
		sessionService: sessionService,
		decision:       decision,
		processor:      proc,
		presenter:      pr,
		persistence:    persistence,
		handoff:        handoffService,
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

	var resp response.Response
	err = w.persistence.WithinMessageTransaction(ctx, func(txCtx context.Context, tx MessageTransaction) error {
		// 2. Save incoming message to DB
		incomingMsg := message.Message{
			SessionID:  sess.ID,
			SenderType: message.SenderTypeUser,
			Text:       msg.Text,
			CreatedAt:  msg.Timestamp,
		}
		if msg.QuickReply != nil && msg.QuickReply.ID != "" {
			quickReplyID := msg.QuickReply.ID
			incomingMsg.Intent = &quickReplyID
		}
		createdMsg, err := tx.CreateMessage(txCtx, incomingMsg)
		if err != nil {
			w.logger.Error("failed to save inbound message",
				w.logger.String("request_id", msg.RequestID),
				w.logger.String("session_id", sess.ID.String()),
				w.logger.String("error_code", string(apperror.CodeDatabaseUnavailable)))
			return apperror.Wrap(apperror.CodeDatabaseUnavailable, "save_message", err)
		}

		if sess.Mode == session.ModeOperatorConnected {
			resp = response.Response{
				SessionID:      sess.ID,
				UserMessageID:  createdMsg.ID,
				Channel:        sess.Channel,
				ExternalUserID: sess.ExternalUserID,
				ClientID:       sess.ClientID,
				ActiveTopic:    sess.ActiveTopic,
				Mode:           sess.Mode,
				OperatorStatus: sess.OperatorStatus,
			}
			return nil
		}

		// 3. Load last 10 messages (now includes the message we just saved)
		history, err := tx.GetLastMessagesBySessionID(txCtx, sess.ID, 10)
		if err != nil {
			w.logger.Error("failed to load message history",
				w.logger.String("request_id", msg.RequestID),
				w.logger.String("session_id", sess.ID.String()),
				w.logger.String("message_id", createdMsg.ID.String()),
				w.logger.String("error_code", string(apperror.CodeDatabaseUnavailable)))
			return apperror.Wrap(apperror.CodeDatabaseUnavailable, "load_message_history", err)
		}

		decisionResult, err := w.decide(txCtx, *sess, history, msg)
		if err != nil {
			w.logger.Error("decision service failed",
				w.logger.String("request_id", msg.RequestID),
				w.logger.String("session_id", sess.ID.String()),
				w.logger.String("message_id", createdMsg.ID.String()),
				w.logger.String("error_code", string(apperror.PublicFromError(err, msg.RequestID).Code)))
			return apperror.Wrap(apperror.CodeProviderUnavailable, "decision_service", err)
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
		actionData.Context["intent"] = decisionResult.Intent
		actionData.Context["state"] = string(decisionResult.State)
		actionData.Context["response_key"] = decisionResult.ResponseKey
		actionData.Context["low_confidence"] = decisionResult.LowConfidence

		actionResults := w.processor.ExecuteWithResults(txCtx, decisionResult.Actions, actionData)

		responseKey := decisionResult.ResponseKey
		if decisionResult.UseActionResponseSelect {
			responseKey, err = w.processor.SelectResponse(txCtx, decisionResult.State, actionResults)
			if err != nil {
				w.logger.Error("failed to select response",
					w.logger.String("request_id", msg.RequestID),
					w.logger.String("session_id", sess.ID.String()),
					w.logger.String("message_id", createdMsg.ID.String()),
					w.logger.String("error_code", string(apperror.CodeProcessingFailed)))
				return apperror.Wrap(apperror.CodeProcessingFailed, "select_response", err)
			}
		}

		handoffReason, shouldQueueHandoff := operatorHandoffReason(decisionResult, actionResults)
		if shouldQueueHandoff && handoffReason == operatorDomain.ReasonBusinessError && responseKey == "" {
			responseKey = "provider_lookup_unavailable"
		}

		if err := tx.LogDecision(txCtx, DecisionLog{
			SessionID:     sess.ID,
			MessageID:     createdMsg.ID,
			Intent:        decisionResult.Intent,
			State:         decisionResult.State,
			ResponseKey:   responseKey,
			Confidence:    decisionResult.Confidence,
			LowConfidence: decisionResult.LowConfidence,
			Candidates:    decisionResult.Candidates,
		}); err != nil {
			w.logger.Error("failed to log decision",
				w.logger.String("request_id", msg.RequestID),
				w.logger.String("session_id", sess.ID.String()),
				w.logger.String("message_id", createdMsg.ID.String()),
				w.logger.String("error_code", string(apperror.CodeDatabaseUnavailable)))
			return apperror.Wrap(apperror.CodeDatabaseUnavailable, "log_decision", err)
		}

		for _, actionName := range decisionResult.Actions {
			if err := tx.LogAction(txCtx, actionLogFromResult(sess.ID, actionName, decisionResult.ActionContext, actionResults[actionName])); err != nil {
				w.logger.Error("failed to log action",
					w.logger.String("request_id", msg.RequestID),
					w.logger.String("session_id", sess.ID.String()),
					w.logger.String("message_id", createdMsg.ID.String()),
					w.logger.String("action", actionName),
					w.logger.String("error_code", string(apperror.CodeDatabaseUnavailable)))
				return apperror.Wrap(apperror.CodeDatabaseUnavailable, "log_action", err)
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
		var updatedSession session.Session
		if shouldQueueHandoff && w.handoff != nil {
			if contextDecision.Metadata == nil {
				contextDecision.Metadata = map[string]interface{}{}
			}
			if handoffReason == operatorDomain.ReasonBusinessError {
				contextDecision.Event = session.EventRequestOperator
				contextDecision.Metadata["handoff_trigger"] = "provider_unavailable"
			}
			snapshot := operatorSnapshot(history, *sess, decisionResult, actionResults)
			item, err := w.handoff.QueueWithDecision(txCtx, sess.ID, handoffReason, snapshot, contextDecision)
			if err != nil {
				w.logger.Error("failed to queue operator handoff",
					w.logger.String("request_id", msg.RequestID),
					w.logger.String("session_id", sess.ID.String()),
					w.logger.String("message_id", createdMsg.ID.String()),
					w.logger.String("reason", string(handoffReason)),
					w.logger.String("error_code", string(apperror.CodeDatabaseUnavailable)))
				return apperror.Wrap(apperror.CodeDatabaseUnavailable, "queue_operator_handoff", err)
			}
			contextDecision.Metadata["handoff_id"] = item.ID.String()
			contextDecision.Metadata["handoff_reason"] = string(handoffReason)
			localSession, _, err := session.PrepareContextUpdate(sess, contextDecision)
			if err != nil {
				w.logger.Error("failed to prepare local handoff context",
					w.logger.String("request_id", msg.RequestID),
					w.logger.String("session_id", sess.ID.String()),
					w.logger.String("message_id", createdMsg.ID.String()),
					w.logger.String("error_code", string(apperror.CodeDatabaseUnavailable)))
				return apperror.Wrap(apperror.CodeDatabaseUnavailable, "prepare_handoff_context", err)
			}
			*sess = localSession
			updatedSession = localSession
		} else {
			updatedSession, err = tx.ApplyContextDecision(txCtx, sess, contextDecision)
			if err != nil {
				w.logger.Error("failed to update session state",
					w.logger.String("request_id", msg.RequestID),
					w.logger.String("session_id", sess.ID.String()),
					w.logger.String("message_id", createdMsg.ID.String()),
					w.logger.String("error_code", string(apperror.CodeDatabaseUnavailable)))
				return apperror.Wrap(apperror.CodeDatabaseUnavailable, "update_session_context", err)
			}
		}

		presented, err := w.presenter.Render(presenter.RenderInput{
			ResponseKey:    responseKey,
			State:          decisionResult.State,
			ActionResults:  actionResults,
			SessionContext: presenter.NewSessionContext(updatedSession),
			Data: map[string]any{
				"intent":       decisionResult.Intent,
				"question":     msg.Text,
				"response_key": responseKey,
			},
		})
		if err != nil {
			w.logger.Error("failed to present response",
				w.logger.String("request_id", msg.RequestID),
				w.logger.String("session_id", sess.ID.String()),
				w.logger.String("message_id", createdMsg.ID.String()),
				w.logger.String("error_code", string(apperror.CodeProcessingFailed)))
			return apperror.Wrap(apperror.CodeProcessingFailed, "present_response", err)
		}

		botMsg := message.Message{
			SessionID:  sess.ID,
			SenderType: message.SenderTypeBot,
			Text:       presented.Text,
			CreatedAt:  time.Now().UTC(),
		}
		createdBotMsg, err := tx.CreateMessage(txCtx, botMsg)
		if err != nil {
			w.logger.Error("failed to save bot message",
				w.logger.String("request_id", msg.RequestID),
				w.logger.String("session_id", sess.ID.String()),
				w.logger.String("message_id", createdMsg.ID.String()),
				w.logger.String("error_code", string(apperror.CodeDatabaseUnavailable)))
			return apperror.Wrap(apperror.CodeDatabaseUnavailable, "save_bot_message", err)
		}

		w.logger.Info("response generated",
			w.logger.String("request_id", msg.RequestID),
			w.logger.String("session_id", sess.ID.String()),
			w.logger.String("message_id", createdBotMsg.ID.String()),
			w.logger.String("response_key", responseKey),
			w.logger.String("state", string(presented.State)),
			w.logger.Int("response_text_length", observability.LenForLog(presented.Text)))

		presented.SessionID = sess.ID
		presented.UserMessageID = createdMsg.ID
		presented.BotMessageID = createdBotMsg.ID
		presented.Channel = updatedSession.Channel
		presented.ExternalUserID = updatedSession.ExternalUserID
		presented.ClientID = updatedSession.ClientID
		presented.ActiveTopic = updatedSession.ActiveTopic
		presented.Mode = updatedSession.Mode
		presented.OperatorStatus = updatedSession.OperatorStatus
		resp = presented
		return nil
	})
	if err != nil {
		if apperror.IsAppError(err) {
			return response.Response{}, err
		}
		return response.Response{}, apperror.Wrap(apperror.CodeDatabaseUnavailable, "message_transaction", err)
	}

	return resp, nil
}

func (w *MessageWorker) decide(
	ctx context.Context,
	sess session.Session,
	history []message.Message,
	msg contracts.IncomingMessage,
) (appdecision.Result, error) {
	if msg.QuickReply == nil {
		return w.decision.Decide(ctx, sess, history, msg.Text)
	}

	quickReplyDecision, ok := w.decision.(QuickReplyDecisionService)
	if !ok {
		return w.decision.Decide(ctx, sess, history, msg.Text)
	}

	return quickReplyDecision.DecideQuickReply(ctx, sess, history, appdecision.QuickReplySelection{
		ID:      msg.QuickReply.ID,
		Action:  msg.QuickReply.Action,
		Payload: cloneQuickReplyPayload(msg.QuickReply.Payload),
	}, msg.Text)
}

func cloneQuickReplyPayload(payload map[string]any) map[string]any {
	if len(payload) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(payload))
	for key, value := range payload {
		cloned[key] = value
	}
	return cloned
}

func actionLogFromResult(
	sessionID uuid.UUID,
	actionName string,
	actionContext map[string]any,
	result processor.ActionResult,
) action.Log {
	responsePayload := make(map[string]interface{})
	if result.Data != nil {
		responsePayload["result"] = result.Data
	}
	if result.Audit != nil {
		responsePayload["audit"] = result.Audit
	}

	var errorValue *string
	if !result.Success && result.Error != "" {
		errCopy := result.Error
		errorValue = &errCopy
	}

	return action.Log{
		SessionID:       sessionID,
		ActionType:      actionName,
		RequestPayload:  safeActionRequestPayload(actionName, actionContext),
		ResponsePayload: responsePayload,
		Error:           errorValue,
	}
}

func safeActionRequestPayload(actionName string, actionContext map[string]any) map[string]interface{} {
	payload := map[string]interface{}{
		"action": actionName,
	}
	if actionContext == nil {
		return payload
	}
	for _, key := range []string{"provided_identifier", "identifier_type", "handoff_reason"} {
		if value, ok := actionContext[key]; ok {
			payload[key] = value
		}
	}
	return payload
}

func operatorHandoffReason(
	decisionResult appdecision.Result,
	actionResults map[string]processor.ActionResult,
) (operatorDomain.Reason, bool) {
	if providerUnavailable(actionResults) {
		return operatorDomain.ReasonBusinessError, true
	}

	if result, ok := actionResults[action.ActionEscalateToOperator]; ok && result.Success {
		if reason := reasonFromActionResult(result); reason != "" {
			return operatorDomain.NormalizeReason(operatorDomain.Reason(reason)), true
		}
		return operatorDomain.ReasonManualRequest, true
	}

	if decisionResult.Event == session.EventRequestOperator {
		if decisionResult.LowConfidence || decisionResult.Intent == "unknown" {
			return operatorDomain.ReasonLowConfidenceRepeated, true
		}
		if decisionResult.Intent == "report_complaint" || hasPrefix(decisionResult.Intent, "complaint_") {
			return operatorDomain.ReasonComplaint, true
		}
		return operatorDomain.ReasonManualRequest, true
	}

	return "", false
}

func providerUnavailable(actionResults map[string]processor.ActionResult) bool {
	for _, result := range actionResults {
		if actionResultStatus(result.Data) == "unavailable" || actionResultStatus(result.Audit) == "unavailable" {
			return true
		}
	}
	return false
}

func reasonFromActionResult(result processor.ActionResult) string {
	if reason := actionResultString(result.Data, "reason"); reason != "" {
		return reason
	}
	if reason := actionResultString(result.Audit, "reason"); reason != "" {
		return reason
	}
	return ""
}

func actionResultStatus(value any) string {
	return actionResultString(value, "status")
}

func actionResultString(value any, key string) string {
	if payload, ok := value.(map[string]any); ok {
		if raw, ok := payload[key].(string); ok {
			return raw
		}
	}
	return ""
}

func operatorSnapshot(
	history []message.Message,
	sess session.Session,
	decisionResult appdecision.Result,
	actionResults map[string]processor.ActionResult,
) operatorDomain.ContextSnapshot {
	activeTopic := firstNonEmptyString(decisionResult.Topic, sess.ActiveTopic)
	lastIntent := firstNonEmptyString(decisionResult.Intent, sess.LastIntent)

	snapshot := operatorDomain.ContextSnapshot{
		LastMessages:    make([]operatorDomain.MessageSnapshot, 0, len(history)),
		ActiveTopic:     activeTopic,
		LastIntent:      lastIntent,
		Confidence:      decisionResult.Confidence,
		FallbackCount:   nextFallbackCount(sess, decisionResult),
		ActionSummaries: make([]operatorDomain.ActionSummary, 0, len(actionResults)),
	}
	for _, item := range history {
		intent := ""
		if item.Intent != nil {
			intent = *item.Intent
		}
		snapshot.LastMessages = append(snapshot.LastMessages, operatorDomain.MessageSnapshot{
			SenderType: string(item.SenderType),
			Text:       item.Text,
			Intent:     intent,
			CreatedAt:  item.CreatedAt.UTC(),
		})
	}
	now := time.Now().UTC()
	for actionName, result := range actionResults {
		status := actionResultStatus(result.Data)
		if status == "" {
			status = actionResultStatus(result.Audit)
		}
		if status == "" && result.Success {
			status = "success"
		}
		if status == "" {
			status = "failed"
		}
		summary := actionResultString(result.Audit, "error_code")
		if summary == "" {
			summary = result.Error
		}
		snapshot.ActionSummaries = append(snapshot.ActionSummaries, operatorDomain.ActionSummary{
			ActionType: actionName,
			Status:     status,
			Summary:    summary,
			CreatedAt:  now,
		})
	}

	return snapshot
}

func nextFallbackCount(sess session.Session, decisionResult appdecision.Result) int {
	if decisionResult.LowConfidence {
		return sess.FallbackCount + 1
	}
	if decisionResult.Intent != "" {
		return 0
	}
	return sess.FallbackCount
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func hasPrefix(value, prefix string) bool {
	return len(value) >= len(prefix) && value[:len(prefix)] == prefix
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

	return nil, session.ErrInvalidIdentity
}
