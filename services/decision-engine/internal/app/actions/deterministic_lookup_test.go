package actions

import (
	"context"
	"path/filepath"
	"testing"

	appseed "github.com/VladKovDev/chat-bot/internal/app/seed"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

func TestFindPaymentUsesSeedFixtureForExactIdentifier(t *testing.T) {
	t.Parallel()

	dataset := mustLoadDataset(t)
	act := NewFindPayment(logger.Noop(), dataset)
	data := action.ActionData{
		Session:  &session.Session{Metadata: map[string]any{}},
		UserText: "Проверь платеж PAY-123456",
		Context: map[string]interface{}{
			"provided_identifier": "PAY-123456",
		},
	}

	if err := act.Execute(context.Background(), data); err != nil {
		t.Fatalf("execute: %v", err)
	}

	result, ok := data.Context["action_result"].(map[string]any)
	if !ok {
		t.Fatalf("action_result type = %T, want map[string]any", data.Context["action_result"])
	}

	if got := result["status"]; got != "found" {
		t.Fatalf("status = %#v, want found", got)
	}
	if got := result["payment_id"]; got != "PAY-123456" {
		t.Fatalf("payment_id = %#v, want PAY-123456", got)
	}
	if got := result["amount"]; got != 2000 {
		t.Fatalf("amount = %#v, want 2000", got)
	}
	if got := result["source"]; got != "mock_external" {
		t.Fatalf("source = %#v, want mock_external", got)
	}
	audit, ok := data.Context["action_audit"].(map[string]any)
	if !ok {
		t.Fatalf("action_audit type = %T, want map[string]any", data.Context["action_audit"])
	}
	for _, key := range []string{"provider", "source", "status", "duration_ms"} {
		if _, exists := audit[key]; !exists {
			t.Fatalf("action_audit missing %q: %#v", key, audit)
		}
	}
}

func TestFindBookingUsesSeedProviderErrorFixture(t *testing.T) {
	t.Parallel()

	dataset := mustLoadDataset(t)
	act := NewFindBooking(logger.Noop(), dataset)
	data := action.ActionData{
		Session:  &session.Session{Metadata: map[string]any{}},
		UserText: "Проверь запись BRG-ERROR-503",
		Context: map[string]interface{}{
			"provided_identifier": "BRG-ERROR-503",
		},
	}

	if err := act.Execute(context.Background(), data); err != nil {
		t.Fatalf("execute: %v", err)
	}

	result, ok := data.Context["action_result"].(map[string]any)
	if !ok {
		t.Fatalf("action_result type = %T, want map[string]any", data.Context["action_result"])
	}
	if got := result["status"]; got != "unavailable" {
		t.Fatalf("status = %#v, want unavailable", got)
	}
	if got := result["error_code"]; got != "provider_unavailable" {
		t.Fatalf("error_code = %#v, want provider_unavailable", got)
	}

	audit, ok := data.Context["action_audit"].(map[string]any)
	if !ok {
		t.Fatalf("action_audit type = %T, want map[string]any", data.Context["action_audit"])
	}
	if got := audit["provider"]; got != "booking" {
		t.Fatalf("provider = %#v, want booking", got)
	}
	if got := audit["status"]; got != "unavailable" {
		t.Fatalf("status = %#v, want unavailable", got)
	}
}

func TestFindPaymentReturnsInvalidStatusForBadIdentifier(t *testing.T) {
	t.Parallel()

	dataset := mustLoadDataset(t)
	act := NewFindPayment(logger.Noop(), dataset)
	data := action.ActionData{
		Session:  &session.Session{Metadata: map[string]any{}},
		UserText: "Проверь платеж PAY",
		Context: map[string]interface{}{
			"provided_identifier": "PAY",
			"identifier_type":     "payment_id",
		},
	}

	if err := act.Execute(context.Background(), data); err != nil {
		t.Fatalf("execute: %v", err)
	}

	result := data.Context["action_result"].(map[string]any)
	if got := result["status"]; got != "invalid" {
		t.Fatalf("status = %#v, want invalid", got)
	}
	if got := result["error_code"]; got != "invalid_identifier" {
		t.Fatalf("error_code = %#v, want invalid_identifier", got)
	}
}

func TestLookupActionsExposeSafeAuditForExpectedProviderFailures(t *testing.T) {
	t.Parallel()

	dataset := mustLoadDataset(t)
	cases := []struct {
		name           string
		execute        func(action.ActionData) error
		identifier     string
		identifierType string
		wantProvider   string
		wantStatus     string
		wantCode       string
	}{
		{
			name: "workspace unavailable",
			execute: func(data action.ActionData) error {
				return NewFindWorkspaceBooking(logger.Noop(), dataset).Execute(context.Background(), data)
			},
			identifier:     "WS-ERROR-503",
			identifierType: "workspace_booking",
			wantProvider:   "workspace_booking",
			wantStatus:     "unavailable",
			wantCode:       "provider_unavailable",
		},
		{
			name: "account invalid",
			execute: func(data action.ActionData) error {
				return NewFindUserAccount(logger.Noop(), dataset).Execute(context.Background(), data)
			},
			identifier:     "broken-email",
			identifierType: "email",
			wantProvider:   "user_account",
			wantStatus:     "invalid",
			wantCode:       "invalid_identifier",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data := action.ActionData{
				Session:  &session.Session{Metadata: map[string]any{}},
				UserText: tc.identifier,
				Context: map[string]interface{}{
					"provided_identifier": tc.identifier,
					"identifier_type":     tc.identifierType,
				},
			}

			if err := tc.execute(data); err != nil {
				t.Fatalf("execute: %v", err)
			}

			result, ok := data.Context["action_result"].(map[string]any)
			if !ok {
				t.Fatalf("action_result type = %T, want map[string]any", data.Context["action_result"])
			}
			if got := result["status"]; got != tc.wantStatus {
				t.Fatalf("status = %#v, want %s", got, tc.wantStatus)
			}
			if got := result["error_code"]; got != tc.wantCode {
				t.Fatalf("error_code = %#v, want %s", got, tc.wantCode)
			}
			if _, exists := result["error"]; exists {
				t.Fatalf("action_result leaked raw error field: %#v", result)
			}

			audit, ok := data.Context["action_audit"].(map[string]any)
			if !ok {
				t.Fatalf("action_audit type = %T, want map[string]any", data.Context["action_audit"])
			}
			if got := audit["provider"]; got != tc.wantProvider {
				t.Fatalf("provider = %#v, want %s", got, tc.wantProvider)
			}
			if got := audit["source"]; got != "mock_external" {
				t.Fatalf("source = %#v, want mock_external", got)
			}
			if got := audit["status"]; got != tc.wantStatus {
				t.Fatalf("status = %#v, want %s", got, tc.wantStatus)
			}
			if got := audit["error_code"]; got != tc.wantCode {
				t.Fatalf("error_code = %#v, want %s", got, tc.wantCode)
			}
			if _, exists := audit["duration_ms"]; !exists {
				t.Fatalf("audit missing duration_ms: %#v", audit)
			}
		})
	}
}

func mustLoadDataset(t *testing.T) *appseed.Dataset {
	t.Helper()

	configPath, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("service root abs: %v", err)
	}

	dataset, err := appseed.Load(configPath)
	if err != nil {
		t.Fatalf("load dataset: %v", err)
	}

	return dataset
}
