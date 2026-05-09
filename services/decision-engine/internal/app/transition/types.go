package transition

import (
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
)

// TransitionConfig represents a single state transition configuration
type TransitionConfig struct {
	From        state.State
	Event       session.Event
	To          state.State
	ResponseKey string
	Actions     []string
}

// GlobalEventConfig represents a global event that can be triggered from any state
type GlobalEventConfig struct {
	Event       session.Event
	To          state.State
	ResponseKey string
	Actions     []string
}

// TransitionResult represents the result of executing a transition
type TransitionResult struct {
	NextState   state.State
	Actions     []string
	ResponseKey string
}