package websocket

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/VladKovDev/web-adapter/internal/client"
	"github.com/VladKovDev/web-adapter/internal/dto"
	"github.com/VladKovDev/web-adapter/pkg/logger"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Handler struct {
	client       *client.Client
	logger       logger.Logger
	now          func() time.Time
	pollInterval time.Duration
}

func NewHandler(client *client.Client, log logger.Logger) *Handler {
	return &Handler{
		client:       client,
		logger:       log,
		now:          func() time.Time { return time.Now().UTC() },
		pollInterval: 250 * time.Millisecond,
	}
}

func (h *Handler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	rawConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("failed to upgrade connection",
			logger.Err(err),
			logger.String("remote_addr", r.RemoteAddr),
		)
		return
	}
	conn := &socketConn{raw: rawConn}
	defer conn.Close()

	h.logger.Info("websocket connection established",
		logger.String("remote_addr", r.RemoteAddr),
	)

	var sessionID string
	var clientID string
	var stopMonitor context.CancelFunc
	defer func() {
		if stopMonitor != nil {
			stopMonitor()
		}
	}()
	state := newSessionRuntimeState()

	for {
		messageType, payload, err := rawConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("websocket connection closed unexpectedly",
					logger.Err(err),
					logger.String("remote_addr", r.RemoteAddr),
				)
			}
			return
		}

		if messageType != websocket.TextMessage {
			h.sendError(conn, sessionID, dto.PublicError{
				Code:      "invalid_request",
				Message:   "Некорректный запрос. Проверьте данные и попробуйте снова.",
				RequestID: newCorrelationID(),
			})
			continue
		}

		var event dto.ClientEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			h.sendError(conn, sessionID, dto.PublicError{
				Code:      "invalid_request",
				Message:   "Некорректный запрос. Проверьте данные и попробуйте снова.",
				RequestID: newCorrelationID(),
			})
			continue
		}

		if event.EventID == "" {
			event.EventID = newCorrelationID()
		}
		if event.CorrelationID == "" {
			event.CorrelationID = event.EventID
		}
		if event.Timestamp == "" {
			event.Timestamp = h.now().Format(time.RFC3339Nano)
		}

		switch event.Type {
		case dto.EventSessionStart:
			clientID = strings.TrimSpace(event.ClientID)
			if clientID == "" {
				h.sendError(conn, sessionID, dto.PublicError{
					Code:      "invalid_request",
					Message:   "Некорректный запрос. Проверьте данные и попробуйте снова.",
					RequestID: event.CorrelationID,
				})
				continue
			}
			startedSessionID, err := h.handleSessionStart(conn, clientID, event)
			if err != nil {
				h.logger.Error("failed to start websocket session", logger.Err(err))
				h.sendError(conn, sessionID, dto.PublicError{
					Code:      "session_start_failed",
					Message:   "Не удалось начать сессию.",
					RequestID: event.CorrelationID,
				})
				continue
			}
			sessionID = startedSessionID
			state.setSessionID(startedSessionID)
			if stopMonitor != nil {
				stopMonitor()
			}
			monitorCtx, cancel := context.WithCancel(context.Background())
			stopMonitor = cancel
			go h.monitorSession(monitorCtx, conn, state)
		case dto.EventMessageUser:
			targetSessionID := firstNonEmpty(strings.TrimSpace(event.SessionID), sessionID)
			if targetSessionID == "" || strings.TrimSpace(event.Text) == "" || clientID == "" {
				h.sendError(conn, targetSessionID, dto.PublicError{
					Code:      "invalid_request",
					Message:   "Некорректный запрос. Проверьте данные и попробуйте снова.",
					RequestID: event.CorrelationID,
				})
				continue
			}
			sessionID = targetSessionID
			if err := h.handleUserMessage(conn, state, clientID, sessionID, event); err != nil {
				h.logger.Error("failed to process websocket user message", logger.Err(err))
			}
		case dto.EventQuickReplySelected:
			targetSessionID := firstNonEmpty(strings.TrimSpace(event.SessionID), sessionID)
			if targetSessionID == "" || event.QuickReply == nil || clientID == "" {
				h.sendError(conn, targetSessionID, dto.PublicError{
					Code:      "invalid_request",
					Message:   "Некорректный запрос. Проверьте данные и попробуйте снова.",
					RequestID: event.CorrelationID,
				})
				continue
			}
			sessionID = targetSessionID
			if err := h.handleQuickReply(conn, state, clientID, sessionID, event); err != nil {
				h.logger.Error("failed to process websocket quick reply", logger.Err(err))
			}
		case dto.EventOperatorClose:
			targetSessionID := firstNonEmpty(strings.TrimSpace(event.SessionID), sessionID)
			if targetSessionID == "" {
				h.sendError(conn, targetSessionID, dto.PublicError{
					Code:      "invalid_request",
					Message:   "Некорректный запрос. Проверьте данные и попробуйте снова.",
					RequestID: event.CorrelationID,
				})
				continue
			}
			sessionID = targetSessionID
			if err := h.handleOperatorClose(conn, state, sessionID, event); err != nil {
				h.logger.Error("failed to close handoff", logger.Err(err))
			}
		default:
			h.sendError(conn, sessionID, dto.PublicError{
				Code:      "invalid_request",
				Message:   "Некорректный запрос. Проверьте данные и попробуйте снова.",
				RequestID: event.CorrelationID,
			})
		}
	}
}

func (h *Handler) handleSessionStart(conn *socketConn, clientID string, event dto.ClientEvent) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	sessionResp, err := h.client.StartSession(ctx, clientID)
	if err != nil {
		return "", err
	}

	sessionEvent := dto.SessionStartedEvent{
		EventEnvelope: dto.EventEnvelope{
			Type:          dto.EventSessionStarted,
			SessionID:     sessionResp.SessionID,
			EventID:       event.EventID,
			CorrelationID: event.CorrelationID,
			Timestamp:     h.now().Format(time.RFC3339Nano),
		},
		Mode:        sessionResp.Mode,
		ActiveTopic: sessionResp.ActiveTopic,
		Resumed:     sessionResp.Resumed,
	}
	return sessionResp.SessionID, h.writeEvent(conn, sessionEvent)
}

func (h *Handler) handleUserMessage(conn *socketConn, state *sessionRuntimeState, clientID string, sessionID string, event dto.ClientEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := h.client.SendMessage(ctx, event.Text, sessionID, clientID, event.EventID)
	if err != nil {
		h.sendError(conn, sessionID, dto.PublicError{
			Code:      "processing_failed",
			Message:   "Не удалось обработать сообщение. Попробуйте позже.",
			RequestID: event.CorrelationID,
		})
		return err
	}

	if err := h.sendBotMessage(conn, resp); err != nil {
		return err
	}
	return h.sendHandoffEvent(conn, state, resp)
}

func (h *Handler) handleQuickReply(conn *socketConn, state *sessionRuntimeState, clientID string, sessionID string, event dto.ClientEvent) error {
	quickReply := event.QuickReply
	if quickReply == nil {
		return fmt.Errorf("quick reply is required")
	}

	if quickReply.Action == "request_operator" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		resp, err := h.client.RequestHandoff(ctx, sessionID)
		if err != nil {
			h.sendError(conn, sessionID, dto.PublicError{
				Code:      "processing_failed",
				Message:   "Не удалось обработать сообщение. Попробуйте позже.",
				RequestID: event.CorrelationID,
			})
			return err
		}
		handoffEvent := dto.HandoffEvent{
			EventEnvelope: dto.EventEnvelope{
				Type:          dto.EventHandoffQueued,
				SessionID:     sessionID,
				EventID:       resp.Handoff.HandoffID,
				CorrelationID: event.CorrelationID,
				Timestamp:     h.now().Format(time.RFC3339Nano),
			},
			Handoff: resp.Handoff,
		}
		if err := h.writeEvent(conn, handoffEvent); err != nil {
			return err
		}
		state.setHandoffStatus(resp.Handoff.Status)
		return nil
	}

	text := quickReply.Label
	if payloadText := quickReplyPayloadText(quickReply.Payload); payloadText != "" {
		text = payloadText
	}
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("quick reply text is empty")
	}

	return h.handleUserMessage(conn, state, clientID, sessionID, dto.ClientEvent{
		Type:          dto.EventMessageUser,
		SessionID:     sessionID,
		EventID:       event.EventID,
		CorrelationID: event.CorrelationID,
		Text:          text,
	})
}

func (h *Handler) handleOperatorClose(conn *socketConn, state *sessionRuntimeState, sessionID string, event dto.ClientEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := h.client.CloseHandoff(ctx, sessionID)
	if err != nil {
		h.sendError(conn, sessionID, dto.PublicError{
			Code:      "processing_failed",
			Message:   "Не удалось обработать сообщение. Попробуйте позже.",
			RequestID: event.CorrelationID,
		})
		return err
	}

	handoffEvent := dto.HandoffEvent{
		EventEnvelope: dto.EventEnvelope{
			Type:          dto.EventHandoffClosed,
			SessionID:     sessionID,
			EventID:       resp.Handoff.HandoffID,
			CorrelationID: event.CorrelationID,
			Timestamp:     h.now().Format(time.RFC3339Nano),
		},
		Handoff: resp.Handoff,
	}
	if err := h.writeEvent(conn, handoffEvent); err != nil {
		return err
	}
	state.setHandoffStatus(resp.Handoff.Status)
	return nil
}

func (h *Handler) sendBotMessage(conn *socketConn, resp dto.DecisionEngineResponse) error {
	return h.writeEvent(conn, dto.MessageBotEvent{
		EventEnvelope: dto.EventEnvelope{
			Type:          dto.EventMessageBot,
			SessionID:     resp.SessionID,
			EventID:       resp.BotMessageID,
			CorrelationID: firstNonEmpty(resp.CorrelationID, resp.BotMessageID),
			Timestamp:     firstNonEmpty(resp.Timestamp, h.now().Format(time.RFC3339Nano)),
		},
		MessageID:    resp.BotMessageID,
		Text:         resp.Text,
		QuickReplies: resp.QuickReplies,
		Mode:         resp.Mode,
		ActiveTopic:  resp.ActiveTopic,
	})
}

func (h *Handler) sendHandoffEvent(conn *socketConn, state *sessionRuntimeState, resp dto.DecisionEngineResponse) error {
	if resp.Handoff == nil {
		return nil
	}

	eventType := dto.EventHandoffQueued
	switch resp.Handoff.Status {
	case "accepted":
		eventType = dto.EventHandoffAccepted
	case "closed":
		eventType = dto.EventHandoffClosed
	}

	handoffEvent := dto.HandoffEvent{
		EventEnvelope: dto.EventEnvelope{
			Type:          eventType,
			SessionID:     resp.SessionID,
			EventID:       resp.Handoff.HandoffID,
			CorrelationID: firstNonEmpty(resp.CorrelationID, resp.Handoff.HandoffID),
			Timestamp:     firstNonEmpty(resp.Timestamp, h.now().Format(time.RFC3339Nano)),
		},
		Handoff: *resp.Handoff,
	}
	if err := h.writeEvent(conn, handoffEvent); err != nil {
		return err
	}
	state.setHandoffStatus(resp.Handoff.Status)
	return nil
}

func (h *Handler) sendError(conn *socketConn, sessionID string, publicError dto.PublicError) {
	errorEvent := dto.ErrorEvent{
		EventEnvelope: dto.EventEnvelope{
			Type:          dto.EventError,
			SessionID:     sessionID,
			EventID:       newCorrelationID(),
			CorrelationID: firstNonEmpty(publicError.RequestID, newCorrelationID()),
			Timestamp:     h.now().Format(time.RFC3339Nano),
		},
		Error: publicError,
	}

	if err := h.writeEvent(conn, errorEvent); err != nil {
		h.logger.Error("failed to send error event", logger.Err(err))
	}
}

func (h *Handler) monitorSession(ctx context.Context, conn *socketConn, state *sessionRuntimeState) {
	ticker := time.NewTicker(h.pollInterval)
	defer ticker.Stop()

	for {
		if err := h.syncSessionEvents(ctx, conn, state); err != nil && ctx.Err() == nil {
			h.logger.Warn("failed to sync session events",
				logger.Err(err),
				logger.String("session_id", state.sessionID()),
			)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (h *Handler) syncSessionEvents(ctx context.Context, conn *socketConn, state *sessionRuntimeState) error {
	sessionID := state.sessionID()
	if sessionID == "" {
		return nil
	}

	pollCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := h.syncHandoffEvents(pollCtx, conn, sessionID, state); err != nil {
		return err
	}
	return h.syncOperatorMessages(pollCtx, conn, sessionID, state)
}

func (h *Handler) syncHandoffEvents(ctx context.Context, conn *socketConn, sessionID string, state *sessionRuntimeState) error {
	for _, status := range []string{"accepted", "closed"} {
		if state.handoffStatus() == status {
			continue
		}

		queue, err := h.client.GetOperatorQueue(ctx, status)
		if err != nil {
			return err
		}

		for _, item := range queue.Items {
			if item.SessionID != sessionID && item.HandoffID != sessionID {
				continue
			}
			if !state.promoteHandoffStatus(status) {
				return nil
			}
			return h.writeEvent(conn, dto.HandoffEvent{
				EventEnvelope: dto.EventEnvelope{
					Type:          handoffEventType(status),
					SessionID:     sessionID,
					EventID:       item.HandoffID,
					CorrelationID: item.HandoffID,
					Timestamp:     firstNonEmpty(item.CreatedAt, h.now().Format(time.RFC3339Nano)),
				},
				Handoff: dto.Handoff{
					HandoffID: item.HandoffID,
					SessionID: item.SessionID,
					Status:    status,
					Reason:    item.Reason,
				},
			})
		}
	}

	return nil
}

func (h *Handler) syncOperatorMessages(ctx context.Context, conn *socketConn, sessionID string, state *sessionRuntimeState) error {
	messages, err := h.client.GetSessionMessages(ctx, sessionID)
	if err != nil {
		return err
	}

	for _, item := range messages.Items {
		if item.SenderType != "operator" {
			continue
		}
		if !state.markOperatorMessageSeen(item.MessageID) {
			continue
		}

		if err := h.writeEvent(conn, dto.MessageOperatorEvent{
			EventEnvelope: dto.EventEnvelope{
				Type:          dto.EventMessageOperator,
				SessionID:     item.SessionID,
				EventID:       item.MessageID,
				CorrelationID: item.MessageID,
				Timestamp:     firstNonEmpty(item.Timestamp, h.now().Format(time.RFC3339Nano)),
			},
			MessageID: item.MessageID,
			Text:      item.Text,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (h *Handler) writeEvent(conn *socketConn, value any) error {
	message, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}
	if err := conn.writeMessage(websocket.TextMessage, message); err != nil {
		return fmt.Errorf("failed to write websocket event: %w", err)
	}
	return nil
}

type socketConn struct {
	raw     *websocket.Conn
	writeMu sync.Mutex
}

func (c *socketConn) Close() error {
	return c.raw.Close()
}

func (c *socketConn) writeMessage(messageType int, payload []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.raw.WriteMessage(messageType, payload)
}

type sessionRuntimeState struct {
	mu                   sync.Mutex
	currentSessionID     string
	currentHandoffStatus string
	operatorMessageIDs   map[string]struct{}
}

func newSessionRuntimeState() *sessionRuntimeState {
	return &sessionRuntimeState{
		operatorMessageIDs: make(map[string]struct{}),
	}
}

func (s *sessionRuntimeState) setSessionID(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentSessionID = sessionID
}

func (s *sessionRuntimeState) sessionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentSessionID
}

func (s *sessionRuntimeState) setHandoffStatus(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentHandoffStatus = status
}

func (s *sessionRuntimeState) handoffStatus() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentHandoffStatus
}

func (s *sessionRuntimeState) promoteHandoffStatus(status string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.currentHandoffStatus == status {
		return false
	}
	s.currentHandoffStatus = status
	return true
}

func (s *sessionRuntimeState) markOperatorMessageSeen(messageID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.operatorMessageIDs[messageID]; exists {
		return false
	}
	s.operatorMessageIDs[messageID] = struct{}{}
	return true
}

func handoffEventType(status string) string {
	switch status {
	case "accepted":
		return dto.EventHandoffAccepted
	case "closed":
		return dto.EventHandoffClosed
	default:
		return dto.EventHandoffQueued
	}
}

func quickReplyPayloadText(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	value, ok := payload["text"]
	if !ok {
		return ""
	}
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func newCorrelationID() string {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return fmt.Sprintf("local-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(data[:])
}
