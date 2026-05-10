package actions

import (
	"context"
	"testing"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
)

func TestEscalateToOperatorStoresSafeResultAndAudit(t *testing.T) {
	t.Parallel()

	data := action.ActionData{
		UserText: "хочу пожаловаться",
		Context: map[string]interface{}{
			"intent": "report_complaint",
		},
	}

	if err := NewEscalateToOperator().Execute(context.Background(), data); err != nil {
		t.Fatalf("execute escalation: %v", err)
	}

	result, ok := data.Context["action_result"].(map[string]any)
	if !ok {
		t.Fatalf("action_result missing: %#v", data.Context)
	}
	if result["reason"] != "complaint" || result["status"] != "queued_requested" {
		t.Fatalf("action_result = %#v, want complaint queued request", result)
	}

	audit, ok := data.Context["action_audit"].(map[string]any)
	if !ok {
		t.Fatalf("action_audit missing: %#v", data.Context)
	}
	if audit["provider"] != "operator_queue" || audit["reason"] != "complaint" {
		t.Fatalf("action_audit = %#v, want operator queue complaint audit", audit)
	}
}
