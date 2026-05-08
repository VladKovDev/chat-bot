package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/VladKovDev/chat-bot/internal/contracts"
	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/pkg/logger"
	"github.com/google/uuid"
)

type DecideRequest struct {
	Text   string `json:"text"`
	ChatID int64  `json:"chat_id,omitempty"`
}

type DecideResponse struct {
	Text    string   `json:"text"`
	Options []string `json:"options,omitempty"`
	State   string   `json:"state"`
	ChatID  int64    `json:"chat_id"`
	Success bool     `json:"success"`
	Error   string   `json:"error,omitempty"`
}

type Handler struct {
	worker MessageHandler
	logger logger.Logger
}

type MessageHandler interface {
	HandleMessage(ctx context.Context, msg contracts.IncomingMessage) (response.Response, error)
}

func NewHandler(worker MessageHandler, logger logger.Logger) *Handler {
	return &Handler{
		worker: worker,
		logger: logger,
	}
}

func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var req DecideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode request", h.logger.Err(err))
		h.respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if req.Text == "" {
		h.respondWithError(w, http.StatusBadRequest, "text field is required")
		return
	}

	// Set default ChatID if not provided
	if req.ChatID == 0 {
		req.ChatID = 1 // Default chat ID for web requests
	}

	// Create incoming message
	incomingMsg := contracts.IncomingMessage{
		EventID:   uuid.New(),
		ChatID:    req.ChatID,
		Text:      req.Text,
		Timestamp: time.Now(),
	}

	// Process message
	resp, err := h.worker.HandleMessage(ctx, incomingMsg)
	if err != nil {
		h.logger.Error("failed to handle message", h.logger.Err(err))
		h.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to process message: %v", err))
		return
	}

	// Respond with success
	h.respondWithSuccess(w, resp.Text, resp.Options, resp.State, req.ChatID)
}

func (h *Handler) respondWithError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(DecideResponse{
		Success: false,
		Error:   message,
	})
}

func (h *Handler) respondWithSuccess(w http.ResponseWriter, text string, options []string, st state.State, chatID int64) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(DecideResponse{
		Text:    text,
		Options: options,
		State:   string(st),
		ChatID:  chatID,
		Success: true,
	})
}
