package processor

import (
	"context"
	"testing"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

func TestResponseSelectorUsesProviderStatusesForFallbacks(t *testing.T) {
	t.Parallel()

	selector := NewResponseSelector(logger.Noop())

	responseKey, err := selector.SelectResponse(context.Background(), state.StateBooking, map[string]ActionResult{
		action.ActionFindBooking: {
			Success: true,
			Data: map[string]any{
				"status": "unavailable",
			},
		},
	})
	if err != nil {
		t.Fatalf("select unavailable response: %v", err)
	}
	if responseKey != "provider_lookup_unavailable" {
		t.Fatalf("response key = %q, want provider_lookup_unavailable", responseKey)
	}

	responseKey, err = selector.SelectResponse(context.Background(), state.StateBooking, map[string]ActionResult{
		action.ActionFindBooking: {
			Success: true,
			Data: map[string]any{
				"status": "invalid",
			},
		},
	})
	if err != nil {
		t.Fatalf("select invalid response: %v", err)
	}
	if responseKey != "error_data_missing" {
		t.Fatalf("response key = %q, want error_data_missing", responseKey)
	}
}
