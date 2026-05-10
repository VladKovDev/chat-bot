package handler

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/VladKovDev/chat-bot/internal/apperror"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/intent"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	httpmiddleware "github.com/VladKovDev/chat-bot/internal/transport/http/middleware"
)

type ConfigLLMResponse struct {
	Data ConfigLLMData `json:"data"`
}

type ConfigLLMData struct {
	Intents []string `json:"intents"`
	States  []string `json:"states"`
	Actions []string `json:"actions"`
}

// ConfigLLM returns the LLM configuration including all available intents, states, and actions
func (h *Handler) ConfigLLM(w http.ResponseWriter, r *http.Request) {
	// Get all intents, states, and actions from domain
	intents := intent.All()
	states := state.All()
	actions := action.All()

	// Build response
	response := ConfigLLMResponse{
		Data: ConfigLLMData{
			Intents: intents,
			States:  states,
			Actions: actions,
		},
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(response); err != nil {
		requestID := httpmiddleware.RequestIDFromRequest(r)
		h.logger.Error("failed to encode config_llm response",
			h.logger.String("request_id", requestID),
			h.logger.String("error_code", string(apperror.CodeInternal)))
		apperror.WriteJSON(w, http.StatusInternalServerError, apperror.NewPublic(apperror.CodeInternal, requestID))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body.Bytes())
}
