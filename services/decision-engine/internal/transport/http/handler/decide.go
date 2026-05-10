package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/VladKovDev/chat-bot/internal/apperror"
	"github.com/VladKovDev/chat-bot/internal/contracts"
	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	httpmiddleware "github.com/VladKovDev/chat-bot/internal/transport/http/middleware"
	"github.com/VladKovDev/chat-bot/pkg/logger"
	"github.com/google/uuid"
)

type DecideRequest struct {
	Text           string `json:"text"`
	SessionID      string `json:"session_id,omitempty"`
	Channel        string `json:"channel,omitempty"`
	ExternalUserID string `json:"external_user_id,omitempty"`
	ClientID       string `json:"client_id,omitempty"`
	ChatID         int64  `json:"chat_id,omitempty"`
}

type DecideResponse struct {
	Text           string                `json:"text"`
	Options        []string              `json:"options,omitempty"`
	State          string                `json:"state"`
	ActiveTopic    string                `json:"active_topic"`
	SessionID      string                `json:"session_id,omitempty"`
	Channel        string                `json:"channel,omitempty"`
	ExternalUserID string                `json:"external_user_id,omitempty"`
	ClientID       string                `json:"client_id,omitempty"`
	ChatID         int64                 `json:"chat_id,omitempty"`
	Success        bool                  `json:"success"`
	Error          *apperror.PublicError `json:"error,omitempty"`
}

type StartSessionRequest struct {
	Channel        string `json:"channel"`
	ExternalUserID string `json:"external_user_id,omitempty"`
	ClientID       string `json:"client_id,omitempty"`
}

type StartSessionResponse struct {
	SessionID      string                `json:"session_id,omitempty"`
	Channel        string                `json:"channel,omitempty"`
	ExternalUserID string                `json:"external_user_id,omitempty"`
	ClientID       string                `json:"client_id,omitempty"`
	State          string                `json:"state,omitempty"`
	ActiveTopic    string                `json:"active_topic"`
	Resumed        bool                  `json:"resumed"`
	Success        bool                  `json:"success"`
	Error          *apperror.PublicError `json:"error,omitempty"`
}

type Handler struct {
	worker MessageHandler
	logger logger.Logger
}

type MessageHandler interface {
	HandleMessage(ctx context.Context, msg contracts.IncomingMessage) (response.Response, error)
	StartSession(ctx context.Context, identity session.Identity) (session.StartResult, error)
}

func NewHandler(worker MessageHandler, logger logger.Logger) *Handler {
	return &Handler{
		worker: worker,
		logger: logger,
	}
}

func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := httpmiddleware.RequestIDFromRequest(r)

	// Parse request
	var req DecideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode request",
			h.logger.String("request_id", requestID),
			h.logger.String("error_code", string(apperror.CodeInvalidRequest)),
		)
		h.respondWithError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	// Validate request
	if req.Text == "" {
		h.respondWithError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	identity := session.Identity{
		Channel:        req.Channel,
		ExternalUserID: req.ExternalUserID,
		ClientID:       req.ClientID,
	}
	identity = session.NormalizeIdentity(identity)

	var sessionID uuid.UUID
	if req.SessionID != "" {
		parsedSessionID, err := uuid.Parse(req.SessionID)
		if err != nil {
			h.respondWithError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
			return
		}
		sessionID = parsedSessionID
	}

	if err := validateDecideIdentity(identity, sessionID, req.ChatID); err != nil {
		h.respondWithError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	// Create incoming message
	incomingMsg := contracts.IncomingMessage{
		EventID:        uuid.New(),
		SessionID:      sessionID,
		ChatID:         req.ChatID,
		Channel:        identity.Channel,
		ExternalUserID: identity.ExternalUserID,
		ClientID:       identity.ClientID,
		Text:           req.Text,
		RequestID:      requestID,
		Timestamp:      time.Now(),
	}

	// Process message
	resp, err := h.worker.HandleMessage(ctx, incomingMsg)
	if err != nil {
		publicError := apperror.PublicFromError(err, requestID)
		h.logger.Error("failed to handle message",
			h.logger.String("request_id", requestID),
			h.logger.String("session_id", req.SessionID),
			h.logger.String("client_id", req.ClientID),
			h.logger.String("channel", req.Channel),
			h.logger.String("error_code", string(publicError.Code)),
		)
		h.respondWithError(w, publicError)
		return
	}

	// Respond with success
	h.respondWithSuccess(w, resp)
}

func (h *Handler) StartSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := httpmiddleware.RequestIDFromRequest(r)

	var req StartSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode session request",
			h.logger.String("request_id", requestID),
			h.logger.String("error_code", string(apperror.CodeInvalidRequest)),
		)
		h.respondWithSessionError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	identity := session.NormalizeIdentity(session.Identity{
		Channel:        req.Channel,
		ExternalUserID: req.ExternalUserID,
		ClientID:       req.ClientID,
	})
	if err := session.ValidateIdentity(identity); err != nil {
		h.respondWithSessionError(w, apperror.NewPublic(apperror.CodeInvalidRequest, requestID))
		return
	}

	result, err := h.worker.StartSession(ctx, identity)
	if err != nil {
		publicError := apperror.PublicFromError(err, requestID)
		h.logger.Error("failed to start session",
			h.logger.String("request_id", requestID),
			h.logger.String("client_id", req.ClientID),
			h.logger.String("channel", req.Channel),
			h.logger.String("error_code", string(publicError.Code)),
		)
		h.respondWithSessionError(w, publicError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(StartSessionResponse{
		SessionID:      result.Session.ID.String(),
		Channel:        result.Session.Channel,
		ExternalUserID: result.Session.ExternalUserID,
		ClientID:       result.Session.ClientID,
		State:          string(result.Session.State),
		ActiveTopic:    result.Session.ActiveTopic,
		Resumed:        result.Resumed,
		Success:        true,
	})
}

func (h *Handler) respondWithError(w http.ResponseWriter, publicError apperror.PublicError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apperror.Status(publicError.Code))
	json.NewEncoder(w).Encode(DecideResponse{
		Success: false,
		Error:   &publicError,
	})
}

func (h *Handler) respondWithSessionError(w http.ResponseWriter, publicError apperror.PublicError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apperror.Status(publicError.Code))
	json.NewEncoder(w).Encode(StartSessionResponse{
		Success: false,
		Error:   &publicError,
	})
}

func (h *Handler) respondWithSuccess(w http.ResponseWriter, resp response.Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(DecideResponse{
		Text:           resp.Text,
		Options:        resp.Options,
		State:          string(resp.State),
		ActiveTopic:    resp.ActiveTopic,
		SessionID:      resp.SessionID.String(),
		Channel:        resp.Channel,
		ExternalUserID: resp.ExternalUserID,
		ClientID:       resp.ClientID,
		Success:        true,
	})
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
