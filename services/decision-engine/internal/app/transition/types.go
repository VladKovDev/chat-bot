package transition

import "github.com/VladKovDev/chat-bot/internal/domain/conversation"

// TransitionConfig represents a single state transition configuration
type TransitionConfig struct {
	From        conversation.State
	Event       conversation.Event
	To          conversation.State
	ResponseKey string
	Actions     []string
}

// GlobalEventConfig represents a global event that can be triggered from any state
type GlobalEventConfig struct {
	Event       conversation.Event
	To          conversation.State
	ResponseKey string
	Actions     []string
}

// TransitionResult represents the result of executing a transition
type TransitionResult struct {
	NextState   conversation.State
	Actions     []string
	ResponseKey string
}