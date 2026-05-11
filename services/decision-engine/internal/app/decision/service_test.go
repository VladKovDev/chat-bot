package decision

import (
	"context"
	"testing"

	apppresenter "github.com/VladKovDev/chat-bot/internal/app/presenter"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
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

func TestDecisionServiceQuickReplySelectIntentUsesPayloadIntentNotLabel(t *testing.T) {
	t.Parallel()

	service, err := NewService(&apppresenter.IntentCatalog{
		Intents: []apppresenter.IntentDefinition{
			{
				Key:            "return_to_menu",
				Category:       "general",
				ResolutionType: "static_response",
				ResponseKey:    "main_menu",
				Examples:       []string{"главное меню"},
			},
		},
	}, stubMatcher{result: MatchResult{}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.DecideQuickReply(
		context.Background(),
		session.Session{},
		nil,
		QuickReplySelection{
			ID:     "renamed-menu",
			Action: "select_intent",
			Payload: map[string]any{
				"intent": "return_to_menu",
			},
		},
		"malicious changed display label",
	)
	if err != nil {
		t.Fatalf("decide quick reply: %v", err)
	}

	if result.Intent != "return_to_menu" {
		t.Fatalf("intent = %q, want return_to_menu", result.Intent)
	}
	if result.ResponseKey != "main_menu" {
		t.Fatalf("response_key = %q, want main_menu", result.ResponseKey)
	}
}

func TestDecisionServiceQuickReplySendTextUsesPayloadText(t *testing.T) {
	t.Parallel()

	service, err := NewService(&apppresenter.IntentCatalog{
		Intents: []apppresenter.IntentDefinition{
			{
				Key:            "ask_workspace_rules",
				Category:       "workspace",
				ResolutionType: "knowledge",
				ResponseKey:    "workspace_rules",
				Examples:       []string{"правила аренды"},
			},
		},
	}, stubMatcher{result: MatchResult{
		IntentKey:  "ask_workspace_rules",
		Confidence: 0.91,
	}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.DecideQuickReply(
		context.Background(),
		session.Session{},
		nil,
		QuickReplySelection{
			ID:     "workspace-rules",
			Action: "send_text",
			Payload: map[string]any{
				"text": "правила аренды",
			},
		},
		"правила аренды",
	)
	if err != nil {
		t.Fatalf("decide quick reply: %v", err)
	}

	if result.Intent != "ask_workspace_rules" {
		t.Fatalf("intent = %q, want ask_workspace_rules", result.Intent)
	}
}

func TestDecisionServiceQuickReplySelectIntentUsesWorkspaceRulesIntent(t *testing.T) {
	t.Parallel()

	service, err := NewService(&apppresenter.IntentCatalog{
		Intents: []apppresenter.IntentDefinition{
			{
				Key:            "ask_workspace_rules",
				Category:       "workspace",
				ResolutionType: "knowledge",
				ResponseKey:    "workspace_rules",
				Examples:       []string{"правила аренды"},
			},
		},
	}, stubMatcher{result: MatchResult{}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.DecideQuickReply(
		context.Background(),
		session.Session{},
		nil,
		QuickReplySelection{
			ID:     "workspace-rules",
			Action: "select_intent",
			Payload: map[string]any{
				"intent": "ask_workspace_rules",
			},
		},
		"правила аренды",
	)
	if err != nil {
		t.Fatalf("decide quick reply: %v", err)
	}

	if result.Intent != "ask_workspace_rules" {
		t.Fatalf("intent = %q, want ask_workspace_rules", result.Intent)
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
	if len(result.Actions) != 1 || result.Actions[0] != action.ActionEscalateToOperator {
		t.Fatalf("actions = %#v, want escalate_to_operator", result.Actions)
	}
	if got := result.ActionContext["handoff_reason"]; got != "low_confidence_repeated" {
		t.Fatalf("handoff_reason = %#v, want low_confidence_repeated", got)
	}
}

func TestDecisionServiceFirstLowConfidenceAsksClarification(t *testing.T) {
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
		session.Session{FallbackCount: 0},
		nil,
		"сложный вопрос",
	)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}

	if result.ResponseKey != "clarify_request" {
		t.Fatalf("response_key = %q, want clarify_request", result.ResponseKey)
	}
	if result.Event != session.EventMessageReceived {
		t.Fatalf("event = %q, want message_received", result.Event)
	}
	if len(result.Actions) != 0 {
		t.Fatalf("actions = %#v, want no escalation on first low confidence", result.Actions)
	}
}

func TestDecisionServiceLowConfidenceAtRootRecoversToStartMenu(t *testing.T) {
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
		IntentKey:      "",
		Confidence:     0.22,
		LowConfidence:  true,
		FallbackReason: defaultLowConfidence,
	}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Decide(
		context.Background(),
		session.Session{Mode: session.ModeStandard},
		nil,
		"привте",
	)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}

	if result.ResponseKey != "start" {
		t.Fatalf("response_key = %q, want start", result.ResponseKey)
	}
	if result.State != state.StateWaitingForCategory {
		t.Fatalf("state = %q, want waiting_for_category", result.State)
	}
	if result.LowConfidence {
		t.Fatalf("LowConfidence = true, want false for root recovery")
	}
}

func TestDecisionServiceLowConfidenceWithActiveTopicStillClarifies(t *testing.T) {
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
		IntentKey:      "",
		Confidence:     0.22,
		LowConfidence:  true,
		FallbackReason: defaultLowConfidence,
	}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Decide(
		context.Background(),
		session.Session{Mode: session.ModeStandard, ActiveTopic: "booking"},
		nil,
		"непонятно",
	)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}

	if result.ResponseKey != "clarify_request" {
		t.Fatalf("response_key = %q, want clarify_request", result.ResponseKey)
	}
	if !result.LowConfidence {
		t.Fatalf("LowConfidence = false, want true for in-topic clarification")
	}
}

func TestDecisionServicePromotesContextualLowConfidenceWithinActiveTopic(t *testing.T) {
	t.Parallel()

	service, err := NewService(&apppresenter.IntentCatalog{
		Intents: []apppresenter.IntentDefinition{
			{
				Key:                 "ask_booking_status",
				Category:            "booking",
				ResolutionType:      "business_lookup",
				ResponseKey:         "booking_request_identifier",
				FallbackResponseKey: "booking_request_identifier",
				Action:              "find_booking",
				Examples:            []string{"статус записи"},
			},
		},
	}, stubMatcher{result: MatchResult{
		IntentKey:      "ask_booking_status",
		Confidence:     0.7335497736930847,
		LowConfidence:  true,
		FallbackReason: defaultLowConfidence,
		Candidates: []Candidate{
			{
				IntentKey:  "ask_booking_status",
				Confidence: 0.7335497736930847,
				Metadata: map[string]any{
					"category": "booking",
				},
			},
		},
	}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Decide(
		context.Background(),
		session.Session{ActiveTopic: "booking"},
		nil,
		"Проверить статус записи",
	)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}

	if result.Intent != "ask_booking_status" {
		t.Fatalf("intent = %q, want ask_booking_status", result.Intent)
	}
	if result.ResponseKey != "booking_request_identifier" {
		t.Fatalf("response_key = %q, want booking_request_identifier", result.ResponseKey)
	}
	if result.State != state.StateWaitingForIdentifier {
		t.Fatalf("state = %q, want waiting_for_identifier", result.State)
	}
}

func TestDecisionServiceDoesNotPromoteContextualLowConfidenceAcrossDifferentTopic(t *testing.T) {
	t.Parallel()

	service, err := NewService(&apppresenter.IntentCatalog{
		Intents: []apppresenter.IntentDefinition{
			{
				Key:                 "ask_booking_status",
				Category:            "booking",
				ResolutionType:      "business_lookup",
				ResponseKey:         "booking_request_identifier",
				FallbackResponseKey: "booking_request_identifier",
				Action:              "find_booking",
				Examples:            []string{"статус записи"},
			},
		},
	}, stubMatcher{result: MatchResult{
		IntentKey:      "ask_booking_status",
		Confidence:     0.7335497736930847,
		LowConfidence:  true,
		FallbackReason: defaultLowConfidence,
		Candidates: []Candidate{
			{
				IntentKey:  "ask_booking_status",
				Confidence: 0.7335497736930847,
				Metadata: map[string]any{
					"category": "booking",
				},
			},
		},
	}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Decide(
		context.Background(),
		session.Session{ActiveTopic: "payment"},
		nil,
		"Проверить статус записи",
	)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}

	if result.Intent != "unknown" {
		t.Fatalf("intent = %q, want unknown", result.Intent)
	}
	if result.ResponseKey != "clarify_request" {
		t.Fatalf("response_key = %q, want clarify_request", result.ResponseKey)
	}
}

func TestDecisionServiceAppliesSemanticThresholdAndAmbiguityPolicy(t *testing.T) {
	t.Parallel()

	catalog := &apppresenter.IntentCatalog{
		Intents: []apppresenter.IntentDefinition{
			{
				Key:            "ask_payment_status",
				Category:       "payment",
				ResolutionType: "business_lookup",
				ResponseKey:    "payment_request_id",
				Action:         "find_payment",
				Examples:       []string{"проверь оплату"},
			},
			{
				Key:            "ask_booking_status",
				Category:       "booking",
				ResolutionType: "business_lookup",
				ResponseKey:    "booking_request_id",
				Action:         "find_booking",
				Examples:       []string{"проверь запись"},
			},
		},
	}

	tests := []struct {
		name  string
		match MatchResult
	}{
		{
			name: "below default semantic threshold",
			match: MatchResult{
				IntentKey:  "ask_payment_status",
				Confidence: 0.77,
				Candidates: []Candidate{
					{IntentKey: "ask_payment_status", Confidence: 0.77, Source: CandidateSourceIntentExample},
				},
			},
		},
		{
			name: "ambiguous candidates",
			match: MatchResult{
				IntentKey:  "ask_payment_status",
				Confidence: 0.90,
				Candidates: []Candidate{
					{IntentKey: "ask_payment_status", Confidence: 0.90, Source: CandidateSourceIntentExample},
					{IntentKey: "ask_booking_status", Confidence: 0.85, Source: CandidateSourceIntentExample},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service, err := NewService(catalog, stubMatcher{result: tt.match}, logger.Noop())
			if err != nil {
				t.Fatalf("new service: %v", err)
			}
			result, err := service.Decide(context.Background(), session.Session{}, nil, "проверь")
			if err != nil {
				t.Fatalf("decide: %v", err)
			}
			if result.Intent != "unknown" || !result.LowConfidence {
				t.Fatalf("result intent/low = %q/%t, want unknown/true", result.Intent, result.LowConfidence)
			}
			if len(result.Candidates) == 0 {
				t.Fatalf("low confidence result lost candidates")
			}
		})
	}
}

func TestExtractIdentifierAcceptsSeedBookingAndWorkspaceIdentifiers(t *testing.T) {
	t.Parallel()

	bookingIdentifier, bookingType := extractIdentifier("Проверь запись BRG-482910", "find_booking")
	if bookingIdentifier != "BRG-482910" || bookingType != "booking_number" {
		t.Fatalf("booking identifier = %q (%q), want BRG-482910 (booking_number)", bookingIdentifier, bookingType)
	}

	workspaceIdentifier, workspaceType := extractIdentifier("Статус брони WS-1001", "find_workspace_booking")
	if workspaceIdentifier != "WS-1001" || workspaceType != "workspace_booking" {
		t.Fatalf("workspace identifier = %q (%q), want WS-1001 (workspace_booking)", workspaceIdentifier, workspaceType)
	}
}
