package conversation

import (
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

// CheckGlobalEvents checks if the given event is a global event that should
// be handled regardless of the current conversation state.
// Global events take precedence over normal state transitions.
//
// Current global events:
// - EventRequestOperator: Can be triggered from any state, escalates to operator
func CheckGlobalEvents(event state.Event, currentState state.State, ctx TransitionContext) GlobalEventResult {
	switch event {
	case state.EventRequestOperator:
		// User requested operator - escalate from any state
		return GlobalEventResult{
			Handled:  true,
			NewState: state.StateEscalatedToOperator,
			Response: response.Response{
				Text:  "Connecting you to a human operator. Please wait...",
				State: state.StateEscalatedToOperator,
			},
		}

	case state.EventOperatorClosed:
		// Operator closed the conversation
		return GlobalEventResult{
			Handled:  true,
			NewState: state.StateClosed,
			Response: response.Response{
				Text:  "The operator has closed this conversation. Thank you for contacting us!",
				State: state.StateClosed,
			},
		}

	default:
		// Not a global event, proceed with normal transition
		return GlobalEventResult{
			Handled: false,
		}
	}
}
