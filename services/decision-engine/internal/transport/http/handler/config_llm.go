package handler

import (
	"encoding/json"
	"net/http"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/intent"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
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

	// Set headers and respond
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode config_llm response", h.logger.Err(err))
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
