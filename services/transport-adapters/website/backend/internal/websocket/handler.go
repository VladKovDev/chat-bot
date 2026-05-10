package websocket

import (
	"context"
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
		h.sendError(conn, "Client identity is required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	sessionResp, err := h.client.StartSession(ctx, clientID)
	cancel()
	if err != nil || !sessionResp.Success || sessionResp.SessionID == "" {
		h.logger.Error("failed to start browser session",
			logger.Err(err),
			logger.String("client_id", clientID),
		)
		h.sendError(conn, "Failed to start session")
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
				logger.Err(err),
				logger.String("remote_addr", r.RemoteAddr),
			)
			h.sendError(conn, "Invalid message format")
			continue
		}

		// Validate message
		if wsMsg.Type != dto.MessageTypeUser {
			h.logger.Warn("invalid message type",
				logger.String("type", wsMsg.Type),
				logger.String("remote_addr", r.RemoteAddr),
			)
			h.sendError(conn, "Invalid message type")
			continue
		}

		if wsMsg.Text == "" {
			h.logger.Warn("empty message text",
				logger.String("remote_addr", r.RemoteAddr),
			)
			h.sendError(conn, "Message text is required")
			continue
		}

		h.logger.Info("processing message",
			logger.String("text", wsMsg.Text),
			logger.String("session_id", sessionID),
			logger.String("client_id", clientID),
		)

		// Send to decision engine
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		resp, err := h.client.SendMessage(ctx, wsMsg.Text, sessionID, clientID)
		cancel()

		if err != nil {
			h.logger.Error("failed to get response from decision engine",
				logger.Err(err),
				logger.String("text", wsMsg.Text),
			)
			h.sendError(conn, "Failed to process message")
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
func (h *Handler) sendError(conn *websocket.Conn, text string) {
	wsErr := dto.WSError{
		Type: dto.MessageTypeError,
		Text: text,
		Code: "processing_error",
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
