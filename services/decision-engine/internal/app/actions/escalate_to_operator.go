package actions

import (
	"context"
	"strings"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
)

// EscalateToOperator escalates the conversation to a human operator
type EscalateToOperator struct{}

// Execute escalates to operator with context
func (a *EscalateToOperator) Execute(ctx context.Context, data action.ActionData) error {
	if data.Context == nil {
		data.Context = map[string]interface{}{}
	}

	reason := handoffReason(data)
	data.Context["handoff_reason"] = reason
	data.Context["action_result"] = map[string]any{
		"status": "queued_requested",
		"reason": reason,
		"source": "operator_queue",
	}
	data.Context["action_audit"] = map[string]any{
		"provider": "operator_queue",
		"source":   "internal",
		"status":   "queued_requested",
		"reason":   reason,
	}

	return nil
}

// NewEscalateToOperator creates a new EscalateToOperator action
func NewEscalateToOperator() *EscalateToOperator {
	return &EscalateToOperator{}
}

func handoffReason(data action.ActionData) string {
	if value, ok := data.Context["handoff_reason"].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}

	intentValue, _ := data.Context["intent"].(string)
	intentValue = strings.TrimSpace(intentValue)
	if intentValue == "unknown" {
		return "low_confidence_repeated"
	}
	if intentValue == "report_complaint" || strings.HasPrefix(intentValue, "complaint_") {
		return "complaint"
	}
	if unavailable, _ := data.Context["provider_unavailable"].(bool); unavailable {
		return "business_error"
	}

	return "manual_request"
}
