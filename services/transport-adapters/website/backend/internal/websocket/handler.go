package websocket

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/VladKovDev/web-adapter/internal/client"
	"github.com/VladKovDev/web-adapter/internal/dto"
	"github.com/VladKovDev/web-adapter/pkg/logger"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for development
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Handler handles WebSocket connections
type Handler struct {
	client *client.Client
	logger logger.Logger
}

// NewHandler creates a new WebSocket handler
func NewHandler(client *client.Client, log logger.Logger) *Handler {
	return &Handler{
		client: client,
		logger: log,
	}
}

// HandleConnection handles a WebSocket connection
func (h *Handler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("failed to upgrade connection",
			logger.Err(err),
			logger.String("remote_addr", r.RemoteAddr),
		)
		return
	}
	defer conn.Close()

	h.logger.Info("websocket connection established",
		logger.String("remote_addr", r.RemoteAddr),
	)

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		h.sendError(conn, dto.PublicError{
			Code:      "invalid_request",
			Message:   "Некорректный запрос. Проверьте данные и попробуйте снова.",
			RequestID: newCorrelationID(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	sessionResp, err := h.client.StartSession(ctx, clientID)
	cancel()
	if err != nil || !sessionResp.Success || sessionResp.SessionID == "" {
		h.logger.Error("failed to start browser session",
			logger.String("error_code", publicErrorCode(sessionResp.Error, "session_start_failed")),
			logger.String("client_id", clientID),
		)
		h.sendError(conn, publicErrorOrDefault(sessionResp.Error, "session_start_failed", "Не удалось начать сессию."))
		return
	}

	if err := h.sendSession(conn, sessionResp); err != nil {
		h.logger.Error("failed to send session handshake",
			logger.Err(err),
			logger.String("remote_addr", r.RemoteAddr),
		)
		return
	}

	sessionID := sessionResp.SessionID

	// Message loop
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("websocket connection closed unexpectedly",
					logger.Err(err),
					logger.String("remote_addr", r.RemoteAddr),
				)
			} else {
				h.logger.Info("websocket connection closed",
					logger.String("remote_addr", r.RemoteAddr),
				)
			}
			break
		}

		// Only accept text messages
		if messageType != websocket.TextMessage {
			h.logger.Warn("received non-text message",
				logger.String("type", fmt.Sprintf("%d", messageType)),
				logger.String("remote_addr", r.RemoteAddr),
			)
			continue
		}

		// Parse message
		var wsMsg dto.WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			h.logger.Error("failed to unmarshal message",
				logger.String("remote_addr", r.RemoteAddr),
			)
			h.sendError(conn, dto.PublicError{
				Code:      "invalid_request",
				Message:   "Некорректный запрос. Проверьте данные и попробуйте снова.",
				RequestID: newCorrelationID(),
			})
			continue
		}

		// Validate message
		if wsMsg.Type != dto.MessageTypeUser {
			h.logger.Warn("invalid message type",
				logger.String("type", wsMsg.Type),
				logger.String("remote_addr", r.RemoteAddr),
			)
			h.sendError(conn, dto.PublicError{
				Code:      "invalid_request",
				Message:   "Некорректный запрос. Проверьте данные и попробуйте снова.",
				RequestID: newCorrelationID(),
			})
			continue
		}

		if wsMsg.Text == "" {
			h.logger.Warn("empty message text",
				logger.String("remote_addr", r.RemoteAddr),
			)
			h.sendError(conn, dto.PublicError{
				Code:      "invalid_request",
				Message:   "Некорректный запрос. Проверьте данные и попробуйте снова.",
				RequestID: newCorrelationID(),
			})
			continue
		}

		h.logger.Info("processing message",
			logger.String("session_id", sessionID),
			logger.String("client_id", clientID),
			logger.Int("text_length", len([]rune(wsMsg.Text))),
		)

		// Send to decision engine
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		resp, err := h.client.SendMessage(ctx, wsMsg.Text, sessionID, clientID)
		cancel()

		if err != nil || !resp.Success {
			publicError := publicErrorOrDefault(resp.Error, "processing_failed", "Не удалось обработать сообщение. Попробуйте позже.")
			h.logger.Error("failed to get response from decision engine",
				logger.String("error_code", publicError.Code),
				logger.String("request_id", publicError.RequestID),
				logger.String("session_id", sessionID),
				logger.String("client_id", clientID),
			)
			h.sendError(conn, publicError)
			continue
		}

		// Send response to client
		if err := h.sendResponse(conn, resp); err != nil {
			h.logger.Error("failed to send response",
				logger.Err(err),
				logger.String("remote_addr", r.RemoteAddr),
			)
			break
		}
	}
}

// sendResponse sends a bot response to the client
func (h *Handler) sendResponse(conn *websocket.Conn, resp dto.DecisionEngineResponse) error {
	wsResp := dto.WSResponse{
		Type:        dto.MessageTypeBot,
		Text:        resp.Text,
		Options:     resp.Options,
		State:       resp.State,
		ActiveTopic: resp.ActiveTopic,
		SessionID:   resp.SessionID,
	}

	message, err := json.Marshal(wsResp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

func (h *Handler) sendSession(conn *websocket.Conn, resp dto.SessionResponse) error {
	wsResp := dto.WSResponse{
		Type:        dto.MessageTypeSession,
		Text:        "",
		State:       resp.State,
		ActiveTopic: resp.ActiveTopic,
		SessionID:   resp.SessionID,
		Resumed:     resp.Resumed,
	}

	message, err := json.Marshal(wsResp)
	if err != nil {
		return fmt.Errorf("failed to marshal session response: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return fmt.Errorf("failed to write session response: %w", err)
	}

	return nil
}

// sendError sends an error message to the client
func (h *Handler) sendError(conn *websocket.Conn, publicError dto.PublicError) {
	wsErr := dto.WSError{
		Type:  dto.MessageTypeError,
		Error: publicError,
	}

	message, err := json.Marshal(wsErr)
	if err != nil {
		h.logger.Error("failed to marshal error message",
			logger.Err(err),
		)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
		h.logger.Error("failed to send error message",
			logger.Err(err),
		)
	}
}

func publicErrorOrDefault(publicError *dto.PublicError, code string, message string) dto.PublicError {
	if publicError != nil {
		return *publicError
	}
	return dto.PublicError{
		Code:      code,
		Message:   message,
		RequestID: newCorrelationID(),
	}
}

func publicErrorCode(publicError *dto.PublicError, fallback string) string {
	if publicError == nil {
		return fallback
	}
	return publicError.Code
}

func newCorrelationID() string {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return fmt.Sprintf("local-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(data[:])
}
