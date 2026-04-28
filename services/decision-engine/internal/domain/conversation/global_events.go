package conversation

import (
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
)

// GlobalEventResult represents the result of a global event check.
// If Handled is true, the event was a global event and the response
// should be used instead of the normal transition.
type GlobalEventResult struct {
	Handled  bool
	NewState state.State
	Response response.Response
}

// GlobalEventHandler is a function that handles global events
type GlobalEventHandler func(ctx HandlerContext) (nextState state.State, responseKey string, err error)

// Global event handlers with response keys
var globalEventHandlers = map[state.Event]GlobalEventHandler{
	state.EventRequestOperator: func(ctx HandlerContext) (state.State, string, error) {
		return state.StateEscalatedToOperator, "escalated_to_operator", nil
	},

	state.EventOperatorClosed: func(ctx HandlerContext) (state.State, string, error) {
		return state.StateClosed, "operator_closed", nil
	},

	// Add more global events here as needed
	// state.EventResolved: func(ctx HandlerContext) (state.State, string, error) {
	//     return state.StateClosed, "resolved", nil
	// },
}

// CheckGlobalEvents checks if the given event is a global event that should
// be handled regardless of the current conversation state.
// Global events take precedence over normal state transitions.
//
// Returns GlobalEventResult with:
// - Handled: true if this was a global event
// - NewState: the new state to transition to
// - Response: the response to send to the user
func CheckGlobalEvents(event state.Event, currentState state.State, ctx HandlerContext) (GlobalEventResult, error) {
	handler, exists := globalEventHandlers[event]
	if !exists {
		// Not a global event, proceed with normal transition
		return GlobalEventResult{Handled: false}, nil
	}

	// Execute global event handler
	nextState, responseKey, err := handler(ctx)
	if err != nil {
		return GlobalEventResult{
			Handled: false,
		}, fmt.Errorf("global event handler error for event %s: %w", event, err)
	}

	// Load response from JSON
	message, options, ok := ctx.ResponseLoader.GetResponse(responseKey)
	if !ok {
		return GlobalEventResult{
			Handled: false,
		}, fmt.Errorf("response key not found for global event %s: %s", event, responseKey)
	}

	return GlobalEventResult{
		Handled:  true,
		NewState: nextState,
		Response: response.Response{
			Text:    message,
			Options: options,
			State:   nextState,
		},
	}, nil
}

// IsGlobalEvent checks if an event is a global event
func IsGlobalEvent(event state.Event) bool {
	_, exists := globalEventHandlers[event]
	return exists
}