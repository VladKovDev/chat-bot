package decision

import (
	"context"
	"testing"

	apppresenter "github.com/VladKovDev/chat-bot/internal/app/presenter"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

type stubMatcher struct {
	result MatchResult
}

func (m stubMatcher) Match(_ context.Context, _ string, _ []apppresenter.IntentDefinition) (MatchResult, error) {
	return m.result, nil
}

func TestDecisionServiceBusinessLookupUsesExtractedIdentifier(t *testing.T) {
	t.Parallel()

	service, err := NewService(&apppresenter.IntentCatalog{
		Intents: []apppresenter.IntentDefinition{
			{
				Key:                 "ask_payment_status",
				Category:            "payment",
				ResolutionType:      "business_lookup",
				ResponseKey:         "payment_request_id",
				FallbackResponseKey: "payment_request_id",
				Action:              "find_payment",
				Examples:            []string{"проверь оплату"},
			},
		},
	}, stubMatcher{result: MatchResult{
		IntentKey:  "ask_payment_status",
		Confidence: 0.92,
	}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Decide(
		context.Background(),
		session.Session{},
		nil,
		"Проверь оплату PAY-123456",
	)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}

	if !result.UseActionResponseSelect {
		t.Fatalf("UseActionResponseSelect = false, want true")
	}
	if result.State != state.StatePayment {
		t.Fatalf("state = %q, want %q", result.State, state.StatePayment)
	}
	if got := result.ActionContext["provided_identifier"]; got != "PAY-123456" {
		t.Fatalf("provided_identifier = %#v, want PAY-123456", got)
	}
	if got := result.ActionContext["identifier_type"]; got != "payment_id" {
		t.Fatalf("identifier_type = %#v, want payment_id", got)
	}
}

func TestDecisionServiceEscalatesAfterRepeatedLowConfidence(t *testing.T) {
	t.Parallel()

	service, err := NewService(&apppresenter.IntentCatalog{
		Intents: []apppresenter.IntentDefinition{
			{
				Key:            "unknown",
				Category:       "fallback",
				ResolutionType: "fallback",
				ResponseKey:    "clarify_request",
				Examples:       []string{"не знаю"},
			},
		},
	}, stubMatcher{result: MatchResult{
		IntentKey:  "",
		Confidence: 0.12,
	}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Decide(
		context.Background(),
		session.Session{FallbackCount: 1},
		nil,
		"сложный вопрос",
	)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}

	if result.ResponseKey != "operator_handoff_requested" {
		t.Fatalf("response_key = %q, want operator_handoff_requested", result.ResponseKey)
	}
	if result.Event != session.EventRequestOperator {
		t.Fatalf("event = %q, want %q", result.Event, session.EventRequestOperator)
	}
	if result.State != state.StateEscalatedToOperator {
		t.Fatalf("state = %q, want %q", result.State, state.StateEscalatedToOperator)
	}
}
