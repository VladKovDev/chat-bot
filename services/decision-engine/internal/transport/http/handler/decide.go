package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/VladKovDev/chat-bot/internal/apperror"
	"github.com/VladKovDev/chat-bot/internal/contracts"
	"github.com/VladKovDev/chat-bot/internal/domain/message"
	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	httpmiddleware "github.com/VladKovDev/chat-bot/internal/transport/http/middleware"
	"github.com/VladKovDev/chat-bot/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const (
	httpMessageTypeUser  = "user_message"
	quickReplyActionSend = "send_text"

	operatorQueueStatusWaiting  = "waiting"
	operatorQueueStatusAccepted = "accepted"
	operatorQueueStatusClosed   = "closed"
)

type MessageHandler interface {
	HandleMessage(ctx context.Context, msg contracts.IncomingMessage) (response.Response, error)
}

type SessionService interface {
	StartSession(ctx context.Context, identity session.Identity) (session.StartResult, error)
	LoadSessionByID(ctx context.Context, sessionID uuid.UUID, identity session.Identity) (*session.Session, error)
	ApplyContextDecision(ctx context.Context, sess *session.Session, decision session.ContextDecision) (session.Session, error)
}

type SessionRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (session.Session, error)
	ListByStatus(ctx context.Context, status session.Status, limit int32, offset int32) ([]session.Session, error)
}

type MessageRepository interface {
	Create(ctx context.Context, msg message.Message) (message.Message, error)
	GetBySessionID(ctx context.Context, sessionID uuid.UUID, limit int32, offset int32) ([]message.Message, error)
	GetLastMessagesBySessionID(ctx context.Context, sessionID uuid.UUID, limit int32) ([]message.Message, error)
}

type ReadinessProvider func(context.Context) ReadyResponse

type StartSessionRequest struct {
	Channel        string `json:"channel"`
	ExternalUserID string `json:"external_user_id,omitempty"`
	ClientID       string `json:"client_id,omitempty"`
}

type StartSessionResponse struct {
	SessionID   string  `json:"session_id"`
	UserID      string  `json:"user_id"`
	Mode        string  `json:"mode"`
	ActiveTopic *string `json:"active_topic"`
	Resumed     bool    `json:"resumed"`
}

type MessageRequest struct {
	Text           string `json:"text"`
	SessionID      string `json:"session_id,omitempty"`
	EventID        string `json:"event_id,omitempty"`
	Type           string `json:"type"`
	Channel        string `json:"channel,omitempty"`
	ExternalUserID string `json:"external_user_id,omitempty"`
	ClientID       string `json:"client_id,omitempty"`
	ChatID         int64  `json:"chat_id,omitempty"`
}

type QuickReply struct {
	ID      string         `json:"id"`
	Label   string         `json:"label"`
	Action  string         `json:"action"`
	Payload map[string]any `json:"payload,omitempty"`
}

type HandoffResponse struct {
	HandoffID  string  `json:"handoff_id"`
	SessionID  string  `json:"session_id"`
	Status     string  `json:"status"`
	Reason     string  `json:"reason,omitempty"`
	OperatorID *string `json:"operator_id,omitempty"`
}

type MessageResponse struct {
	SessionID     string            `json:"session_id"`
	UserMessageID string            `json:"user_message_id"`
	BotMessageID  string            `json:"bot_message_id"`
	Mode          string            `json:"mode"`
	ActiveTopic   *string           `json:"active_topic"`
	Text          string            `json:"text"`
	QuickReplies  []QuickReply      `json:"quick_replies,omitempty"`
	Handoff       *HandoffResponse  `json:"handoff"`
	CorrelationID string            `json:"correlation_id"`
	Timestamp     string            `json:"timestamp"`
}

type SessionMessagesResponse struct {
	Items []SessionMessageRecord `json:"items"`
}

type SessionMessageRecord struct {
	MessageID  string  `json:"message_id"`
	SessionID  string  `json:"session_id"`
	SenderType string  `json:"sender_type"`
	Text       string  `json:"text"`
	Intent     *string `json:"intent,omitempty"`
	Timestamp  string  `json:"timestamp"`
}

type DomainSchemaResponse struct {
	Intents           []string       `json:"intents"`
	States            []string       `json:"states"`
	Actions           []string       `json:"actions"`
	Channels          []string       `json:"channels"`
	Modes             []string       `json:"modes"`
	OperatorStatuses  []string       `json:"operator_statuses"`
	QuickReplyActions []string       `json:"quick_reply_actions"`
	WebSocketEvents   WebSocketEvent `json:"websocket_events"`
}

type WebSocketEvent struct {
	Client []string `json:"client"`
	Server []string `json:"server"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type ReadyResponse struct {
	Ready     bool                     `json:"ready"`
	Timestamp string                   `json:"timestamp"`
	Checks    map[string]ReadinessItem `json:"checks"`
}

type ReadinessItem struct {
	Ready   bool   `json:"ready"`
	Message string `json:"message,omitempty"`
}

type RequestOperatorBody struct {
	Reason string `json:"reason,omitempty"`
}

type OperatorQueueResponse struct {
	Items []OperatorQueueItem `json:"items"`
}

type OperatorQueueItem struct {
	HandoffID   string  `json:"handoff_id"`
	SessionID   string  `json:"session_id"`
	Reason      string  `json:"reason"`
	ActiveTopic *string `json:"active_topic"`
	LastIntent  *string `json:"last_intent"`
	CreatedAt   string  `json:"created_at"`
	Preview     string  `json:"preview"`
}

type OperatorQueueActionRequest struct {
	OperatorID string `json:"operator_id"`
}

type OperatorQueueActionResponse struct {
	Handoff HandoffResponse `json:"handoff"`
}

type OperatorMessageRequest struct {
	OperatorID string `json:"operator_id"`
	Text       string `json:"text"`
}

type OperatorMessageResponse struct {
	SessionID     string `json:"session_id"`
	MessageID     string `json:"message_id"`
	OperatorID    string `json:"operator_id"`
	Text          string `json:"text"`
	CorrelationID string `json:"correlation_id"`
	Timestamp     string `json:"timestamp"`
}

type Handler struct {
	worker    MessageHandler
	sessions  SessionService
	sessionDB SessionRepository
	messages  MessageRepository
	logger    logger.Logger
	now       func() time.Time
	ready     ReadinessProvider
}

func NewHandler(
	worker MessageHandler,
	sessions SessionService,
	sessionDB SessionRepository,
	messages MessageRepository,
	logger logger.Logger,
) *Handler {
	return &Handler{
		worker:    worker,
		sessions:  sessions,
		sessionDB: sessionDB,
		messages:  messages,
		logger:    logger,
		now:       func() time.Time { return time.Now().UTC() },
		ready:     defaultReadiness,
	}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	h.respondJSON(w, http.StatusOK, HealthResponse{
		Status:    "ok",
		Timestamp: h.now().Format(time.RFC3339Nano),
	})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	resp := h.ready(r.Context())
	status := http.StatusOK
	if !resp.Ready {
		status = http.StatusServiceUnavailable
	}
	h.respondJSON(w, status, resp)
}

func (h *Handler) StartSession(w http.ResponseWriter, r *http.Request) {
	requestID := httpmiddleware.RequestIDFromRequest(r)

	var req StartSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	identity := session.NormalizeIdentity(session.Identity{
		Channel:        req.Channel,
		ExternalUserID: req.ExternalUserID,
		ClientID:       req.ClientID,
	})
	if err := session.ValidateIdentity(identity); err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	result, err := h.sessions.StartSession(r.Context(), identity)
	if err != nil {
		h.respondWithPublicError(w, apperror.PublicFromError(err, requestID))
		return
	}

	h.respondJSON(w, http.StatusOK, StartSessionResponse{
		SessionID:   result.Session.ID.String(),
		UserID:      result.Session.UserID.String(),
		Mode:        string(result.Session.Mode),
		ActiveTopic: optionalString(result.Session.ActiveTopic),
		Resumed:     result.Resumed,
	})
}

func (h *Handler) Message(w http.ResponseWriter, r *http.Request) {
	requestID := httpmiddleware.RequestIDFromRequest(r)

	var req MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	if strings.TrimSpace(req.Text) == "" || req.Type != httpMessageTypeUser {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	identity := session.NormalizeIdentity(session.Identity{
		Channel:        req.Channel,
		ExternalUserID: req.ExternalUserID,
		ClientID:       req.ClientID,
	})

	var sessionID uuid.UUID
	if req.SessionID != "" {
		parsedSessionID, err := uuid.Parse(req.SessionID)
		if err != nil {
			h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
			return
		}
		sessionID = parsedSessionID
	}

	if err := validateDecideIdentity(identity, sessionID, req.ChatID); err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	eventID := uuid.New()
	if req.EventID != "" {
		parsedEventID, err := uuid.Parse(req.EventID)
		if err != nil {
			h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
			return
		}
		eventID = parsedEventID
	}

	resp, err := h.worker.HandleMessage(r.Context(), contracts.IncomingMessage{
		EventID:        eventID,
		SessionID:      sessionID,
		ChatID:         req.ChatID,
		Channel:        identity.Channel,
		ExternalUserID: identity.ExternalUserID,
		ClientID:       identity.ClientID,
		Text:           strings.TrimSpace(req.Text),
		RequestID:      requestID,
		Timestamp:      h.now(),
	})
	if err != nil {
		h.respondWithPublicError(w, apperror.PublicFromError(err, requestID))
		return
	}

	h.respondJSON(w, http.StatusOK, MessageResponse{
		SessionID:     resp.SessionID.String(),
		UserMessageID: resp.UserMessageID.String(),
		BotMessageID:  resp.BotMessageID.String(),
		Mode:          string(resp.Mode),
		ActiveTopic:   optionalString(resp.ActiveTopic),
		Text:          resp.Text,
		QuickReplies:  buildQuickReplies(resp.Options),
		Handoff:       buildHandoff(resp),
		CorrelationID: requestID,
		Timestamp:     h.now().Format(time.RFC3339Nano),
	})
}

func (h *Handler) SessionMessages(w http.ResponseWriter, r *http.Request) {
	requestID := httpmiddleware.RequestIDFromRequest(r)
	sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
	if err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	items, err := h.messages.GetBySessionID(r.Context(), sessionID, 100, 0)
	if err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeDatabaseUnavailable, requestID))
		return
	}

	resp := SessionMessagesResponse{
		Items: make([]SessionMessageRecord, 0, len(items)),
	}
	for _, item := range items {
		resp.Items = append(resp.Items, SessionMessageRecord{
			MessageID:  item.ID.String(),
			SessionID:  item.SessionID.String(),
			SenderType: string(item.SenderType),
			Text:       item.Text,
			Intent:     item.Intent,
			Timestamp:  item.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}

	h.respondJSON(w, http.StatusOK, resp)
}

func (h *Handler) RequestOperator(w http.ResponseWriter, r *http.Request) {
	requestID := httpmiddleware.RequestIDFromRequest(r)
	sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
	if err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	body := RequestOperatorBody{}
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err.Error() != "EOF" {
			h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
			return
		}
	}

	sess, err := h.sessionDB.GetByID(r.Context(), sessionID)
	if err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeDatabaseUnavailable, requestID))
		return
	}

	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		reason = "manual_request"
	}

	updated, err := h.sessions.ApplyContextDecision(r.Context(), &sess, session.ContextDecision{
		Event: session.EventRequestOperator,
		Metadata: map[string]interface{}{
			"handoff_reason": reason,
		},
	})
	if err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeDatabaseUnavailable, requestID))
		return
	}

	h.respondJSON(w, http.StatusOK, OperatorQueueActionResponse{
		Handoff: HandoffResponse{
			HandoffID: updated.ID.String(),
			SessionID: updated.ID.String(),
			Status:    operatorQueueStatusWaiting,
			Reason:    reason,
		},
	})
}

func (h *Handler) OperatorQueue(w http.ResponseWriter, r *http.Request) {
	requestID := httpmiddleware.RequestIDFromRequest(r)
	queueStatus := strings.TrimSpace(r.URL.Query().Get("status"))
	if queueStatus == "" {
		queueStatus = operatorQueueStatusWaiting
	}

	var desiredSessionStatus session.Status
	if queueStatus == operatorQueueStatusClosed {
		desiredSessionStatus = session.StatusClosed
	} else {
		desiredSessionStatus = session.StatusActive
	}

	sessions, err := h.sessionDB.ListByStatus(r.Context(), desiredSessionStatus, 100, 0)
	if err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeDatabaseUnavailable, requestID))
		return
	}

	items := make([]OperatorQueueItem, 0, len(sessions))
	for _, item := range sessions {
		if !matchesQueueStatus(item, queueStatus) {
			continue
		}
		preview := ""
		lastMessages, err := h.messages.GetLastMessagesBySessionID(r.Context(), item.ID, 1)
		if err == nil && len(lastMessages) > 0 {
			preview = lastMessages[0].Text
		}
		items = append(items, OperatorQueueItem{
			HandoffID:   item.ID.String(),
			SessionID:   item.ID.String(),
			Reason:      metadataString(item.Metadata, "handoff_reason", "manual_request"),
			ActiveTopic: optionalString(item.ActiveTopic),
			LastIntent:  optionalString(item.LastIntent),
			CreatedAt:   queueCreatedAt(item).Format(time.RFC3339Nano),
			Preview:     preview,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt < items[j].CreatedAt
	})

	h.respondJSON(w, http.StatusOK, OperatorQueueResponse{Items: items})
}

func (h *Handler) AcceptOperatorQueue(w http.ResponseWriter, r *http.Request) {
	requestID := httpmiddleware.RequestIDFromRequest(r)
	sessionID, err := uuid.Parse(chi.URLParam(r, "handoff_id"))
	if err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	var req OperatorQueueActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.OperatorID) == "" {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	sess, err := h.sessionDB.GetByID(r.Context(), sessionID)
	if err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeDatabaseUnavailable, requestID))
		return
	}

	updated, err := h.sessions.ApplyContextDecision(r.Context(), &sess, session.ContextDecision{
		Event: session.EventOperatorConnected,
		Metadata: map[string]interface{}{
			"operator_id": strings.TrimSpace(req.OperatorID),
		},
	})
	if err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeDatabaseUnavailable, requestID))
		return
	}

	operatorID := strings.TrimSpace(req.OperatorID)
	h.respondJSON(w, http.StatusOK, OperatorQueueActionResponse{
		Handoff: HandoffResponse{
			HandoffID:  updated.ID.String(),
			SessionID:  updated.ID.String(),
			Status:     operatorQueueStatusAccepted,
			Reason:     metadataString(updated.Metadata, "handoff_reason", "manual_request"),
			OperatorID: &operatorID,
		},
	})
}

func (h *Handler) OperatorMessage(w http.ResponseWriter, r *http.Request) {
	requestID := httpmiddleware.RequestIDFromRequest(r)
	sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
	if err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	var req OperatorMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.OperatorID) == "" || strings.TrimSpace(req.Text) == "" {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	if _, err := h.sessionDB.GetByID(r.Context(), sessionID); err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeDatabaseUnavailable, requestID))
		return
	}

	created, err := h.messages.Create(r.Context(), message.Message{
		SessionID:  sessionID,
		SenderType: message.SenderTypeOperator,
		Text:       strings.TrimSpace(req.Text),
		CreatedAt:  h.now(),
	})
	if err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeDatabaseUnavailable, requestID))
		return
	}

	h.respondJSON(w, http.StatusOK, OperatorMessageResponse{
		SessionID:     sessionID.String(),
		MessageID:     created.ID.String(),
		OperatorID:    strings.TrimSpace(req.OperatorID),
		Text:          created.Text,
		CorrelationID: requestID,
		Timestamp:     created.CreatedAt.UTC().Format(time.RFC3339Nano),
	})
}

func (h *Handler) CloseOperatorQueue(w http.ResponseWriter, r *http.Request) {
	requestID := httpmiddleware.RequestIDFromRequest(r)
	sessionID, err := uuid.Parse(chi.URLParam(r, "handoff_id"))
	if err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	req := OperatorQueueActionRequest{}
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
			h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
			return
		}
	}

	sess, err := h.sessionDB.GetByID(r.Context(), sessionID)
	if err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeDatabaseUnavailable, requestID))
		return
	}

	updated, err := h.sessions.ApplyContextDecision(r.Context(), &sess, session.ContextDecision{
		Event: session.EventOperatorClosed,
		Metadata: map[string]interface{}{
			"operator_id": strings.TrimSpace(req.OperatorID),
		},
	})
	if err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeDatabaseUnavailable, requestID))
		return
	}

	var operatorID *string
	if value := strings.TrimSpace(req.OperatorID); value != "" {
		operatorID = &value
	}

	h.respondJSON(w, http.StatusOK, OperatorQueueActionResponse{
		Handoff: HandoffResponse{
			HandoffID:  updated.ID.String(),
			SessionID:  updated.ID.String(),
			Status:     operatorQueueStatusClosed,
			Reason:     metadataString(updated.Metadata, "handoff_reason", "manual_request"),
			OperatorID: operatorID,
		},
	})
}

func (h *Handler) respondWithPublicError(w http.ResponseWriter, publicError apperror.PublicError) {
	apperror.WriteJSON(w, apperror.Status(publicError.Code), publicError)
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func buildQuickReplies(options []string) []QuickReply {
	if len(options) == 0 {
		return nil
	}

	result := make([]QuickReply, 0, len(options))
	for _, option := range options {
		label := strings.TrimSpace(option)
		if label == "" {
			continue
		}
		result = append(result, QuickReply{
			ID:     slugifyQuickReplyID(label),
			Label:  label,
			Action: quickReplyActionSend,
			Payload: map[string]any{
				"text": label,
			},
		})
	}
	return result
}

func buildHandoff(resp response.Response) *HandoffResponse {
	status := handoffStatus(resp.Mode)
	if status == "" {
		return nil
	}

	return &HandoffResponse{
		HandoffID: resp.SessionID.String(),
		SessionID: resp.SessionID.String(),
		Status:    status,
	}
}

func handoffStatus(mode session.Mode) string {
	switch mode {
	case session.ModeWaitingOperator:
		return operatorQueueStatusWaiting
	case session.ModeOperatorConnected:
		return operatorQueueStatusAccepted
	case session.ModeClosed:
		return operatorQueueStatusClosed
	default:
		return ""
	}
}

func matchesQueueStatus(sess session.Session, desired string) bool {
	switch desired {
	case operatorQueueStatusWaiting:
		return sess.OperatorStatus == session.OperatorStatusWaiting
	case operatorQueueStatusAccepted:
		return sess.OperatorStatus == session.OperatorStatusConnected
	case operatorQueueStatusClosed:
		return sess.OperatorStatus == session.OperatorStatusClosed || sess.Status == session.StatusClosed
	default:
		return false
	}
}

func metadataString(metadata map[string]interface{}, key, fallback string) string {
	if metadata == nil {
		return fallback
	}
	raw, ok := metadata[key]
	if !ok {
		return fallback
	}
	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func queueCreatedAt(sess session.Session) time.Time {
	if !sess.UpdatedAt.IsZero() {
		return sess.UpdatedAt.UTC()
	}
	if !sess.CreatedAt.IsZero() {
		return sess.CreatedAt.UTC()
	}
	return time.Unix(0, 0).UTC()
}

func defaultReadiness(_ context.Context) ReadyResponse {
	return ReadyResponse{
		Ready:     false,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Checks: map[string]ReadinessItem{
			"database": {
				Ready:   true,
				Message: "database connection initialized",
			},
			"migrations": {
				Ready:   false,
				Message: "migrations readiness probe is not implemented yet",
			},
			"nlp": {
				Ready:   false,
				Message: "nlp service readiness is not wired yet",
			},
			"pgvector": {
				Ready:   false,
				Message: "vector index readiness is not wired yet",
			},
			"seed_data": {
				Ready:   false,
				Message: "seed data readiness is not wired yet",
			},
		},
	}
}

func validateDecideIdentity(identity session.Identity, sessionID uuid.UUID, chatID int64) error {
	if identity.Channel == session.ChannelDevCLI && chatID != 0 && identity.ExternalUserID == "" && identity.ClientID == "" {
		return nil
	}

	if sessionID != uuid.Nil || identity.Channel != "" || identity.ExternalUserID != "" || identity.ClientID != "" {
		return session.ValidateIdentity(identity)
	}

	if chatID != 0 {
		return errDevCLIChannelRequired()
	}

	return session.ErrInvalidIdentity
}

func errDevCLIChannelRequired() error {
	return identityError("chat_id is only accepted with channel dev-cli")
}

type identityError string

func (e identityError) Error() string {
	return string(e)
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func slugifyQuickReplyID(label string) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(label) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if lastDash {
			continue
		}
		builder.WriteByte('-')
		lastDash = true
	}
	id := strings.Trim(builder.String(), "-")
	if id == "" {
		return "quick-reply"
	}
	return id
}
