package handler

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/VladKovDev/chat-bot/internal/apperror"
	"github.com/VladKovDev/chat-bot/internal/contracts"
	domaindialogreset "github.com/VladKovDev/chat-bot/internal/domain/dialogreset"
	"github.com/VladKovDev/chat-bot/internal/domain/message"
	operatorDomain "github.com/VladKovDev/chat-bot/internal/domain/operator"
	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	httpmiddleware "github.com/VladKovDev/chat-bot/internal/transport/http/middleware"
	"github.com/VladKovDev/chat-bot/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const (
	httpMessageTypeUser       = "user_message"
	httpMessageTypeQuickReply = "quick_reply.selected"
	quickReplyActionSend      = "send_text"

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

type OperatorQueueService interface {
	Queue(ctx context.Context, sessionID uuid.UUID, reason operatorDomain.Reason, snapshot operatorDomain.ContextSnapshot) (operatorDomain.QueueItem, error)
	Accept(ctx context.Context, queueID uuid.UUID, operatorID string) (operatorDomain.QueueItem, error)
	Close(ctx context.Context, queueID uuid.UUID, operatorID string) (operatorDomain.QueueItem, error)
	ListByStatus(ctx context.Context, status operatorDomain.QueueStatus, limit int32, offset int32) ([]operatorDomain.QueueItem, error)
}

type DialogResetRequest = domaindialogreset.Request

type DialogResetSummary = domaindialogreset.Summary

type DialogResetter interface {
	ResetSession(ctx context.Context, req DialogResetRequest) (DialogResetSummary, error)
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
	Text           string      `json:"text"`
	SessionID      string      `json:"session_id,omitempty"`
	EventID        string      `json:"event_id,omitempty"`
	Type           string      `json:"type"`
	Channel        string      `json:"channel,omitempty"`
	ExternalUserID string      `json:"external_user_id,omitempty"`
	ClientID       string      `json:"client_id,omitempty"`
	QuickReply     *QuickReply `json:"quick_reply,omitempty"`
}

type QuickReply struct {
	ID      string         `json:"id"`
	Label   string         `json:"label"`
	Action  string         `json:"action"`
	Payload map[string]any `json:"payload,omitempty"`
	Order   int            `json:"order,omitempty"`
}

type HandoffResponse struct {
	HandoffID  string  `json:"handoff_id"`
	SessionID  string  `json:"session_id"`
	Status     string  `json:"status"`
	Reason     string  `json:"reason,omitempty"`
	OperatorID *string `json:"operator_id,omitempty"`
}

type MessageResponse struct {
	SessionID     string           `json:"session_id"`
	UserMessageID string           `json:"user_message_id"`
	BotMessageID  string           `json:"bot_message_id"`
	Mode          string           `json:"mode"`
	ActiveTopic   *string          `json:"active_topic"`
	Text          string           `json:"text"`
	QuickReplies  []QuickReply     `json:"quick_replies,omitempty"`
	Handoff       *HandoffResponse `json:"handoff"`
	CorrelationID string           `json:"correlation_id"`
	Timestamp     string           `json:"timestamp"`
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
	HandoffID       string                  `json:"handoff_id"`
	SessionID       string                  `json:"session_id"`
	Status          string                  `json:"status"`
	Reason          string                  `json:"reason"`
	OperatorID      *string                 `json:"operator_id,omitempty"`
	ActiveTopic     *string                 `json:"active_topic"`
	LastIntent      *string                 `json:"last_intent"`
	Confidence      *float64                `json:"confidence,omitempty"`
	FallbackCount   int                     `json:"fallback_count"`
	ActionSummaries []OperatorActionSummary `json:"action_summaries"`
	CreatedAt       string                  `json:"created_at"`
	Preview         string                  `json:"preview"`
}

type OperatorActionSummary struct {
	ActionType string `json:"action_type"`
	Status     string `json:"status"`
	Summary    string `json:"summary,omitempty"`
	CreatedAt  string `json:"created_at"`
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

type SessionResetRequest struct {
	Reason string `json:"reason,omitempty"`
}

type SessionResetResponse struct {
	SessionID string           `json:"session_id"`
	Existed   bool             `json:"existed"`
	Deleted   map[string]int64 `json:"deleted"`
	AuditID   string           `json:"audit_id"`
	Timestamp string           `json:"timestamp"`
}

type Handler struct {
	worker     MessageHandler
	sessions   SessionService
	sessionDB  SessionRepository
	messages   MessageRepository
	operators  OperatorQueueService
	resetter   DialogResetter
	adminToken string
	logger     logger.Logger
	now        func() time.Time
	ready      ReadinessProvider
}

func NewHandler(
	worker MessageHandler,
	sessions SessionService,
	sessionDB SessionRepository,
	messages MessageRepository,
	logger logger.Logger,
	operators ...OperatorQueueService,
) *Handler {
	var operatorQueue OperatorQueueService
	if len(operators) > 0 {
		operatorQueue = operators[0]
	}
	return &Handler{
		worker:    worker,
		sessions:  sessions,
		sessionDB: sessionDB,
		messages:  messages,
		operators: operatorQueue,
		logger:    logger,
		now:       func() time.Time { return time.Now().UTC() },
		ready:     defaultReadiness,
	}
}

func (h *Handler) SetDialogResetter(resetter DialogResetter, adminToken string) {
	h.resetter = resetter
	h.adminToken = strings.TrimSpace(adminToken)
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	h.respondJSON(w, http.StatusOK, HealthResponse{
		Status:    "ok",
		Timestamp: h.now().Format(time.RFC3339Nano),
	})
}

func (h *Handler) SetReadiness(provider ReadinessProvider) {
	if provider != nil {
		h.ready = provider
	}
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
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	messageText := strings.TrimSpace(req.Text)
	quickReply, err := normalizeQuickReplyMessage(req.Type, messageText, req.QuickReply)
	if err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}
	if quickReply != nil {
		messageText = quickReplyMessageText(messageText, quickReply)
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

	if err := validateDecideIdentity(identity, sessionID); err != nil {
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
		Channel:        identity.Channel,
		ExternalUserID: identity.ExternalUserID,
		ClientID:       identity.ClientID,
		Text:           messageText,
		QuickReply:     quickReply,
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
		QuickReplies:  buildQuickReplies(resp.QuickReplies, resp.Options),
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
	if h.operators == nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeDatabaseUnavailable, requestID))
		return
	}
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

	snapshot := h.buildOperatorSnapshot(r.Context(), sess)
	item, err := h.operators.Queue(r.Context(), sessionID, operatorDomain.Reason(reason), snapshot)
	if err != nil {
		h.respondOperatorError(w, err, requestID)
		return
	}

	h.respondJSON(w, http.StatusOK, OperatorQueueActionResponse{
		Handoff: handoffResponseFromQueue(item),
	})
}

func (h *Handler) OperatorQueue(w http.ResponseWriter, r *http.Request) {
	requestID := httpmiddleware.RequestIDFromRequest(r)
	if h.operators == nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeDatabaseUnavailable, requestID))
		return
	}
	queueStatus := strings.TrimSpace(r.URL.Query().Get("status"))
	if queueStatus == "" {
		queueStatus = operatorQueueStatusWaiting
	}

	queueItems, err := h.operators.ListByStatus(r.Context(), operatorDomain.QueueStatus(queueStatus), 100, 0)
	if err != nil {
		h.respondOperatorError(w, err, requestID)
		return
	}

	items := make([]OperatorQueueItem, 0, len(queueItems))
	for _, item := range queueItems {
		items = append(items, queueItemResponseFromDomain(item))
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt < items[j].CreatedAt
	})

	h.respondJSON(w, http.StatusOK, OperatorQueueResponse{Items: items})
}

func (h *Handler) AcceptOperatorQueue(w http.ResponseWriter, r *http.Request) {
	requestID := httpmiddleware.RequestIDFromRequest(r)
	if h.operators == nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeDatabaseUnavailable, requestID))
		return
	}
	handoffID, err := uuid.Parse(chi.URLParam(r, "handoff_id"))
	if err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	var req OperatorQueueActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.OperatorID) == "" {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	item, err := h.operators.Accept(r.Context(), handoffID, req.OperatorID)
	if err != nil {
		h.respondOperatorError(w, err, requestID)
		return
	}

	h.respondJSON(w, http.StatusOK, OperatorQueueActionResponse{
		Handoff: handoffResponseFromQueue(item),
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
	if h.operators == nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeDatabaseUnavailable, requestID))
		return
	}
	handoffID, err := uuid.Parse(chi.URLParam(r, "handoff_id"))
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

	item, err := h.operators.Close(r.Context(), handoffID, req.OperatorID)
	if err != nil {
		h.respondOperatorError(w, err, requestID)
		return
	}

	h.respondJSON(w, http.StatusOK, OperatorQueueActionResponse{
		Handoff: handoffResponseFromQueue(item),
	})
}

func (h *Handler) ResetSession(w http.ResponseWriter, r *http.Request) {
	requestID := httpmiddleware.RequestIDFromRequest(r)
	if !h.authorizedAdmin(r) {
		apperror.WriteJSON(w, http.StatusForbidden, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}
	if h.resetter == nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeDatabaseUnavailable, requestID))
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
	if err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	body := SessionResetRequest{}
	if r.Body != nil {
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
			h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
			return
		}
	}

	summary, err := h.resetter.ResetSession(r.Context(), DialogResetRequest{
		SessionID: sessionID,
		Actor:     "admin_http",
		Reason:    strings.TrimSpace(body.Reason),
	})
	if err != nil {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeDatabaseUnavailable, requestID))
		return
	}

	h.respondJSON(w, http.StatusOK, SessionResetResponse{
		SessionID: summary.SessionID.String(),
		Existed:   summary.Existed,
		Deleted:   summary.Deleted,
		AuditID:   summary.AuditID.String(),
		Timestamp: summary.CreatedAt.UTC().Format(time.RFC3339Nano),
	})
}

func (h *Handler) authorizedAdmin(r *http.Request) bool {
	expected := strings.TrimSpace(h.adminToken)
	if expected == "" {
		return false
	}
	actual := strings.TrimSpace(r.Header.Get("X-Admin-Token"))
	if actual == "" || len(actual) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}

func (h *Handler) respondWithPublicError(w http.ResponseWriter, publicError apperror.PublicError) {
	apperror.WriteJSON(w, apperror.Status(publicError.Code), publicError)
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func buildQuickReplies(configured []response.QuickReply, options []string) []QuickReply {
	if len(configured) > 0 {
		result := make([]QuickReply, 0, len(configured))
		for _, quickReply := range configured {
			if strings.TrimSpace(quickReply.Label) == "" {
				continue
			}
			result = append(result, QuickReply{
				ID:      quickReply.ID,
				Label:   quickReply.Label,
				Action:  quickReply.Action,
				Payload: clonePayload(quickReply.Payload),
				Order:   quickReply.Order,
			})
		}
		if len(result) > 0 {
			return result
		}
	}
	_ = options
	return nil
}

func normalizeQuickReplyMessage(eventType string, messageText string, quickReply *QuickReply) (*contracts.QuickReplySelection, error) {
	switch eventType {
	case httpMessageTypeUser:
		if strings.TrimSpace(messageText) == "" || quickReply != nil {
			return nil, errors.New("invalid user message")
		}
		return nil, nil
	case httpMessageTypeQuickReply:
		if quickReply == nil {
			return nil, errors.New("quick reply is required")
		}
		id := strings.TrimSpace(quickReply.ID)
		action := strings.TrimSpace(quickReply.Action)
		if id == "" || action == "" || strings.TrimSpace(quickReply.Label) == "" {
			return nil, errors.New("quick reply id, label and action are required")
		}
		switch action {
		case quickReplyActionSend:
			if quickReplyPayloadString(quickReply.Payload, "text") == "" {
				return nil, errors.New("quick reply send_text payload.text is required")
			}
		case "select_intent":
			if quickReplyPayloadString(quickReply.Payload, "intent") == "" {
				return nil, errors.New("quick reply select_intent payload.intent is required")
			}
		case "request_operator":
		default:
			return nil, errors.New("unsupported quick reply action")
		}
		return &contracts.QuickReplySelection{
			ID:      id,
			Label:   strings.TrimSpace(quickReply.Label),
			Action:  action,
			Payload: clonePayload(quickReply.Payload),
		}, nil
	default:
		return nil, errors.New("unsupported message type")
	}
}

func quickReplyMessageText(messageText string, quickReply *contracts.QuickReplySelection) string {
	if messageText != "" {
		return messageText
	}
	if text := quickReplyPayloadString(quickReply.Payload, "text"); text != "" {
		return text
	}
	if intentKey := quickReplyPayloadString(quickReply.Payload, "intent"); intentKey != "" {
		return intentKey
	}
	return quickReply.Action
}

func quickReplyPayloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok {
		return ""
	}
	text, _ := value.(string)
	return strings.TrimSpace(text)
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

func (h *Handler) buildOperatorSnapshot(ctx context.Context, sess session.Session) operatorDomain.ContextSnapshot {
	lastMessages, err := h.messages.GetLastMessagesBySessionID(ctx, sess.ID, 20)
	if err != nil {
		lastMessages = nil
	}
	sort.Slice(lastMessages, func(i, j int) bool {
		return lastMessages[i].CreatedAt.Before(lastMessages[j].CreatedAt)
	})

	snapshot := operatorDomain.ContextSnapshot{
		LastMessages:    make([]operatorDomain.MessageSnapshot, 0, len(lastMessages)),
		ActiveTopic:     sess.ActiveTopic,
		LastIntent:      sess.LastIntent,
		FallbackCount:   sess.FallbackCount,
		ActionSummaries: []operatorDomain.ActionSummary{},
	}
	for _, item := range lastMessages {
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
	return snapshot
}

func queueItemResponseFromDomain(item operatorDomain.QueueItem) OperatorQueueItem {
	resp := OperatorQueueItem{
		HandoffID:       item.ID.String(),
		SessionID:       item.SessionID.String(),
		Status:          string(item.Status),
		Reason:          string(item.Reason),
		ActiveTopic:     optionalString(item.ContextSnapshot.ActiveTopic),
		LastIntent:      optionalString(item.ContextSnapshot.LastIntent),
		Confidence:      item.ContextSnapshot.Confidence,
		FallbackCount:   item.ContextSnapshot.FallbackCount,
		ActionSummaries: operatorActionSummariesFromDomain(item.ContextSnapshot.ActionSummaries),
		CreatedAt:       item.CreatedAt.UTC().Format(time.RFC3339Nano),
		Preview:         queuePreview(item.ContextSnapshot),
	}
	if strings.TrimSpace(item.AssignedOperatorID) != "" {
		operatorID := item.AssignedOperatorID
		resp.OperatorID = &operatorID
	}
	return resp
}

func operatorActionSummariesFromDomain(items []operatorDomain.ActionSummary) []OperatorActionSummary {
	if len(items) == 0 {
		return []OperatorActionSummary{}
	}
	resp := make([]OperatorActionSummary, 0, len(items))
	for _, item := range items {
		resp = append(resp, OperatorActionSummary{
			ActionType: item.ActionType,
			Status:     item.Status,
			Summary:    item.Summary,
			CreatedAt:  item.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return resp
}

func queuePreview(snapshot operatorDomain.ContextSnapshot) string {
	for i := len(snapshot.LastMessages) - 1; i >= 0; i-- {
		if text := strings.TrimSpace(snapshot.LastMessages[i].Text); text != "" {
			return text
		}
	}
	return ""
}

func handoffResponseFromQueue(item operatorDomain.QueueItem) HandoffResponse {
	resp := HandoffResponse{
		HandoffID: item.ID.String(),
		SessionID: item.SessionID.String(),
		Status:    string(item.Status),
		Reason:    string(item.Reason),
	}
	if strings.TrimSpace(item.AssignedOperatorID) != "" {
		operatorID := item.AssignedOperatorID
		resp.OperatorID = &operatorID
	}
	return resp
}

func (h *Handler) respondOperatorError(w http.ResponseWriter, err error, requestID string) {
	if errors.Is(err, operatorDomain.ErrInvalidReason) ||
		errors.Is(err, operatorDomain.ErrInvalidStatus) ||
		errors.Is(err, operatorDomain.ErrInvalidOperator) ||
		errors.Is(err, operatorDomain.ErrInvalidTransition) {
		h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}
	h.respondWithPublicError(w, apperror.NewPublic(apperror.CodeDatabaseUnavailable, requestID))
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

func validateDecideIdentity(identity session.Identity, sessionID uuid.UUID) error {
	if sessionID != uuid.Nil || identity.Channel != "" || identity.ExternalUserID != "" || identity.ClientID != "" {
		return session.ValidateIdentity(identity)
	}

	return session.ErrInvalidIdentity
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

func clonePayload(payload map[string]any) map[string]any {
	if len(payload) == 0 {
		return nil
	}

	cloned := make(map[string]any, len(payload))
	for key, value := range payload {
		cloned[key] = value
	}
	return cloned
}
