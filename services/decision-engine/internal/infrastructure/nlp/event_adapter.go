package nlp

import (
	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
	"github.com/VladKovDev/chat-bot/internal/domain/intent"
)

// EventAdapter maps user intents to system events
type EventAdapter struct {
	intentToEvent map[intent.Intent]conversation.Event
}

// NewEventAdapter creates a new EventAdapter with the given intent-to-event mapping
func NewEventAdapter(intentToEvent map[intent.Intent]conversation.Event) *EventAdapter {
	return &EventAdapter{
		intentToEvent: intentToEvent,
	}
}

// IntentToEvent converts a user intent to a system event
// Returns EventMessageReceived as default if no mapping found
func (a *EventAdapter) IntentToEvent(intent intent.Intent) conversation.Event {
	if event, ok := a.intentToEvent[intent]; ok {
		return event
	}
	return conversation.EventMessageReceived // default
}