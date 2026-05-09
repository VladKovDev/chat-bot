package processor

import (
	"context"

	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

// ActionResult represents the result of an action execution
type ActionResult struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ResponseSelector selects the appropriate response based on state and action results
type ResponseSelector struct {
	logger logger.Logger
}

func NewResponseSelector(logger logger.Logger) *ResponseSelector {
	return &ResponseSelector{
		logger: logger,
	}
}

// SelectResponse determines the response key based on state and action results
func (rs *ResponseSelector) SelectResponse(
	ctx context.Context,
	currentState state.State,
	actionResults map[string]ActionResult,
) (string, error) {
	rs.logger.Debug("selecting response",
		rs.logger.String("state", string(currentState)),
		rs.logger.Int("actions", len(actionResults)))

	// Strategy 1: Check for critical action failures
	if failedAction := rs.getFailedCriticalAction(actionResults); failedAction != "" {
		return rs.getErrorResponseKey(failedAction), nil
	}

	// Strategy 2: State-based response selection
	responseKey := rs.getStateResponseKey(currentState, actionResults)

	rs.logger.Info("response selected",
		rs.logger.String("response_key", responseKey),
		rs.logger.String("state", string(currentState)))

	return responseKey, nil
}

// getFailedCriticalAction checks if any critical action failed
func (rs *ResponseSelector) getFailedCriticalAction(
	results map[string]ActionResult,
) string {
	criticalActions := map[string]bool{
		"escalate_operator": true,
		"create_ticket":     true,
		"process_payment":   true,
	}

	for actionName, result := range results {
		if !result.Success && criticalActions[actionName] {
			return actionName
		}
	}
	return ""
}

// getErrorResponseKey returns error response key for failed action
func (rs *ResponseSelector) getErrorResponseKey(action string) string {
	errorResponses := map[string]string{
		"escalate_operator": "operator_escalation_failed",
		"create_ticket":     "ticket_creation_failed",
		"process_payment":   "payment_failed",
	}

	if key, ok := errorResponses[action]; ok {
		return key
	}
	return "error_occurred"
}

// getStateResponseKey selects response based on state and action results
func (rs *ResponseSelector) getStateResponseKey(
	currentState state.State,
	actionResults map[string]ActionResult,
) string {
	// Special state handling
	switch currentState {
	case state.StateEscalatedToOperator:
		return "escalated_to_operator"

	case state.StateNew:
		if _, ok := actionResults["reset_conversation"]; ok {
			return "start"
		}
		return "greeting"

	case state.StateClosed:
		return "conversation_closed"

	// Booking states
	case "waiting_booking_date":
		if rs.allActionsSucceeded(actionResults) {
			return "booking_confirmed"
		}
		return "booking_failed"

	// Payment states
	case "waiting_payment":
		if result, ok := actionResults["process_payment"]; ok {
			if result.Success {
				return "payment_success"
			}
			return "payment_failed"
		}
		return "payment_pending"

	// Default: use intent-based or generic response
	default:
		return rs.getDefaultResponseKey(currentState, actionResults)
	}
}

// getDefaultResponseKey returns default response for state
func (rs *ResponseSelector) getDefaultResponseKey(
	currentState state.State,
	actionResults map[string]ActionResult,
) string {
	// Check if any specific action dictates response
	for actionName := range actionResults {
		if responseKey, ok := rs.getActionResponseKey(actionName); ok {
			return responseKey
		}
	}

	// Fallback to state-based response
	stateResponses := map[state.State]string{
		"waiting_for_category":    "ask_category",
		"waiting_clarification":   "ask_clarification",
		"solution_offered":        "solution_explained",
		"waiting_booking_details": "ask_booking_details",
	}

	if key, ok := stateResponses[currentState]; ok {
		return key
	}

	return "default_response"
}

// getActionResponseKey returns response key based on executed action
func (rs *ResponseSelector) getActionResponseKey(action string) (string, bool) {
	actionResponses := map[string]string{
		"reset_conversation": "conversation_reset",
		"escalate_operator":  "escalated_to_operator",
		"create_ticket":      "ticket_created",
		"save_category":      "category_saved",
		"save_context":       "context_saved",
		"send_confirmation":  "confirmation_sent",
	}

	key, ok := actionResponses[action]
	return key, ok
}

// allActionsSucceeded checks if all actions completed successfully
func (rs *ResponseSelector) allActionsSucceeded(results map[string]ActionResult) bool {
	for _, result := range results {
		if !result.Success {
			return false
		}
	}
	return true
}