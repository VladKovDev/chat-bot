package transition

import (
	"github.com/VladKovDev/chat-bot/internal/domain/session"
)

// TransitionConfig represents a single state transition configuration
type TransitionConfig struct {
	From        session.Mode
	Event       session.Event
	To          session.Mode
	ResponseKey string
	Actions     []string
}

// GlobalEventConfig represents a global event that can be triggered from any state
type GlobalEventConfig struct {
	Event       session.Event
	To          session.Mode
	ResponseKey string
	Actions     []string
}

// TransitionResult represents the result of executing a transition
type TransitionResult struct {
	NextMode    session.Mode
	Actions     []string
	ResponseKey string
}
