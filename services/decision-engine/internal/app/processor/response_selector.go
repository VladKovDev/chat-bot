package processor

import (
	"context"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
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
// using a priority-based hierarchy to ensure correct response selection
func (rs *ResponseSelector) SelectResponse(
	ctx context.Context,
	currentState state.State,
	actionResults map[string]ActionResult,
) (string, error) {
	rs.logger.Debug("selecting response",
		rs.logger.String("state", string(currentState)),
		rs.logger.Int("actions", len(actionResults)))

	// Priority 1: Check for validation failures
	if key := rs.checkValidationFailures(actionResults); key != "" {
		rs.logger.Debug("response selected: validation failure",
			rs.logger.String("response_key", key))
		return key, nil
	}

	// Priority 2: Check data action results
	if key := rs.checkDataActionResults(actionResults); key != "" {
		rs.logger.Debug("response selected: data action result",
			rs.logger.String("response_key", key))
		return key, nil
	}

	// Priority 3: Check escalation actions
	if key := rs.checkEscalationActions(currentState, actionResults); key != "" {
		rs.logger.Debug("response selected: escalation action",
			rs.logger.String("response_key", key))
		return key, nil
	}

	// Priority 4: Check terminal states
	if key := rs.checkTerminalStates(currentState, actionResults); key != "" {
		rs.logger.Debug("response selected: terminal state",
			rs.logger.String("response_key", key))
		return key, nil
	}

	// Priority 5: Check category states
	if key := rs.checkCategoryStates(currentState); key != "" {
		rs.logger.Debug("response selected: category state",
			rs.logger.String("response_key", key))
		return key, nil
	}

	// Priority 6: Check information/contact states
	if key := rs.checkInformationStates(currentState); key != "" {
		rs.logger.Debug("response selected: information state",
			rs.logger.String("response_key", key))
		return key, nil
	}

	// Priority 7: Check waiting states
	if key := rs.checkWaitingStates(currentState); key != "" {
		rs.logger.Debug("response selected: waiting state",
			rs.logger.String("response_key", key))
		return key, nil
	}

	// Priority 8: Final fallback
	return rs.getFallbackResponse(currentState)
}

// checkValidationFailures checks if validation actions failed
// Priority 1: Critical validation failures that should prevent further processing
func (rs *ResponseSelector) checkValidationFailures(
	results map[string]ActionResult,
) string {
	// Check validate_identifier action
	if result, ok := results[action.ActionValidateIdentifier]; ok {
		if !result.Success {
			return "error_data_missing"
		}
		// Check if validation returned valid=false
		if data, ok := result.Data.(map[string]interface{}); ok {
			if valid, ok := data["valid"].(bool); ok && !valid {
				return "error_data_missing"
			}
		}
	}
	return ""
}

// checkDataActionResults checks results from data-finding actions
// Priority 2: Data search results (found/not_found) should influence response
func (rs *ResponseSelector) checkDataActionResults(
	results map[string]ActionResult,
) string {
	actionMappings := map[string]map[string]string{
		action.ActionFindBooking: {
			"found":     "booking_found",
			"not_found": "booking_not_found",
		},
		action.ActionFindWorkspaceBooking: {
			"found":     "workspace_booking_found",
			"not_found": "workspace_booking_not_found",
		},
		action.ActionFindPayment: {
			"found":     "payment_found",
			"not_found": "payment_not_found",
		},
		action.ActionFindUserAccount: {
			"found":     "account_found",
			"not_found": "account_not_found",
		},
	}

	for actionName, statusMapping := range actionMappings {
		if result, ok := results[actionName]; ok && result.Success {
			if status := rs.extractStatus(result.Data); status != "" {
				if responseKey, ok := statusMapping[status]; ok {
					return responseKey
				}
			}
		}
	}
	return ""
}

// checkEscalationActions checks escalation-related actions and states
// Priority 3: Escalation actions should override normal state responses
func (rs *ResponseSelector) checkEscalationActions(
	currentState state.State,
	results map[string]ActionResult,
) string {
	// Check if escalate_to_operator was executed
	if result, ok := results[action.ActionEscalateToOperator]; ok {
		if result.Success {
			return "escalation_context_sent"
		} else {
			return "error_generic"
		}
	}

	// Check if already in escalated state
	if currentState == state.StateEscalatedToOperator {
		return "escalation_to_operator"
	}

	return ""
}

// checkTerminalStates checks for conversation-ending states
// Priority 4: Terminal states should override category/information states
func (rs *ResponseSelector) checkTerminalStates(
	currentState state.State,
	results map[string]ActionResult,
) string {
	switch currentState {
	case state.StateClosed:
		return "goodbye"

	case state.StateNew:
		if _, ok := results[action.ActionResetConversation]; ok {
			return "start"
		}
		return "greeting"
	}
	return ""
}

// checkCategoryStates checks if current state is a category state
// Priority 5: Category states should show category-specific menus
func (rs *ResponseSelector) checkCategoryStates(
	currentState state.State,
) string {
	categoryMapping := map[state.State]string{
		state.StateBooking:     "booking_category",
		state.StateWorkspace:   "workspace_category",
		state.StatePayment:     "payment_category",
		state.StateTechIssue:   "tech_issue_category",
		state.StateAccount:     "account_category",
		state.StateServices:    "services_category",
		state.StateComplaint:   "complaint_category",
		state.StateOther:       "other_category",
	}

	if key, ok := categoryMapping[currentState]; ok {
		return key
	}
	return ""
}

// checkInformationStates checks for information-providing states
// Priority 6: Information states show specific information responses
func (rs *ResponseSelector) checkInformationStates(
	currentState state.State,
) string {
	switch currentState {
	case state.StateShowContactInfo:
		return "contact_info"
	}
	return ""
}

// checkWaitingStates checks for waiting/input states
// Priority 7: Waiting states request input from the user
func (rs *ResponseSelector) checkWaitingStates(
	currentState state.State,
) string {
	switch currentState {
	case state.StateWaitingForCategory:
		return "main_menu"
	case state.StateWaitingClarification:
		return "clarify_request"
	case state.StateWaitingForIdentifier:
		return "error_data_missing" // Will be context-aware in future
	}
	return ""
}

// getFallbackResponse provides a safe fallback when no other response matches
// Priority 8: Always returns a valid response key
func (rs *ResponseSelector) getFallbackResponse(
	currentState state.State,
) (string, error) {
	rs.logger.Warn("using fallback response",
		rs.logger.String("state", string(currentState)))

	// error_generic always exists in responses.json
	return "error_generic", nil
}

// extractStatus extracts the status field from action result data
func (rs *ResponseSelector) extractStatus(
	data interface{},
) string {
	if data == nil {
		return ""
	}

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return ""
	}

	if status, ok := dataMap["status"].(string); ok {
		return status
	}

	return ""
}
