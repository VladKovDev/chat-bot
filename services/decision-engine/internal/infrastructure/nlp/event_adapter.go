package nlp

import (
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/intent"
)

// EventAdapter maps user intents to system events
type EventAdapter struct {
	intentToEvent map[intent.Intent]session.Event
}

// NewEventAdapter creates a new EventAdapter with the given intent-to-event mapping
func NewEventAdapter(intentToEvent map[intent.Intent]session.Event) *EventAdapter {
	return &EventAdapter{
		intentToEvent: intentToEvent,
	}
}

// IntentToEvent converts a user intent to a system event
// Returns EventMessageReceived as default if no mapping found
func (a *EventAdapter) IntentToEvent(intent intent.Intent) session.Event {
	if event, ok := a.intentToEvent[intent]; ok {
		return event
	}
	return session.EventMessageReceived // default
}