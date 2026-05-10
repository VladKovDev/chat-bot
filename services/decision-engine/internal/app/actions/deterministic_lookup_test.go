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

	err := act.Execute(context.Background(), data)
	if err == nil {
		t.Fatal("expected provider error")
	}

	providerErr, ok := err.(appseed.ProviderError)
	if !ok {
		t.Fatalf("error type = %T, want seed.ProviderError", err)
	}
	if providerErr.Provider != "booking" {
		t.Fatalf("provider = %q, want booking", providerErr.Provider)
	}
	if providerErr.Code != "provider_unavailable" {
		t.Fatalf("code = %q, want provider_unavailable", providerErr.Code)
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
