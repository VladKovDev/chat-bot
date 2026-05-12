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

type stubKnowledgeSearcher struct {
	candidate *Candidate
	err       error
	calls     int
}

func (m stubMatcher) Match(_ context.Context, _ string, _ []apppresenter.IntentDefinition) (MatchResult, error) {
	return m.result, nil
}

func (s *stubKnowledgeSearcher) Retrieve(
	_ context.Context,
	_ string,
	_ apppresenter.IntentDefinition,
) (*Candidate, error) {
	s.calls++
	return s.candidate, s.err
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
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates = %#v, want one quick reply evidence candidate", result.Candidates)
	}
	if result.Candidates[0].Source != CandidateSourceQuickReplyIntent || result.Candidates[0].Confidence != 1 {
		t.Fatalf("candidate = %#v, want quick_reply_intent confidence=1", result.Candidates[0])
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
	if len(result.Candidates) != 1 || result.Candidates[0].Source != CandidateSourceQuickReplyIntent {
		t.Fatalf("candidates = %#v, want quick_reply_intent evidence", result.Candidates)
	}
}

func TestDecisionServiceQuickReplyRequestOperatorUsesDeterministicEvidence(t *testing.T) {
	t.Parallel()

	service, err := NewService(&apppresenter.IntentCatalog{
		Intents: []apppresenter.IntentDefinition{
			{
				Key:            "request_operator",
				Category:       "operator",
				ResolutionType: "operator_handoff",
				ResponseKey:    "operator_handoff_requested",
				Examples:       []string{"оператор"},
			},
		},
	}, stubMatcher{result: MatchResult{}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.DecideQuickReply(
		context.Background(),
		session.Session{ActiveTopic: "payment"},
		nil,
		QuickReplySelection{
			ID:      "operator-now",
			Action:  "request_operator",
			Payload: map[string]any{},
		},
		"",
	)
	if err != nil {
		t.Fatalf("decide quick reply: %v", err)
	}

	if result.Intent != "request_operator" {
		t.Fatalf("intent = %q, want request_operator", result.Intent)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].Source != CandidateSourceQuickReplyIntent {
		t.Fatalf("candidates = %#v, want quick_reply_intent evidence", result.Candidates)
	}
	if result.Candidates[0].IntentKey != "request_operator" || result.Candidates[0].Confidence != 1 {
		t.Fatalf("candidate = %#v, want request_operator confidence=1", result.Candidates[0])
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

func TestDecisionServiceLowConfidenceAtRootClarifies(t *testing.T) {
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

	if result.ResponseKey != "clarify_request" {
		t.Fatalf("response_key = %q, want clarify_request", result.ResponseKey)
	}
	if result.State != state.StateWaitingClarification {
		t.Fatalf("state = %q, want waiting_clarification", result.State)
	}
	if !result.LowConfidence {
		t.Fatalf("LowConfidence = false, want true")
	}
}

func TestDecisionServiceVeryLowConfidenceAtRootClarifies(t *testing.T) {
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
		Confidence:     0.18,
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
		"совсем непонятный вопрос",
	)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}

	if result.ResponseKey != "clarify_request" {
		t.Fatalf("response_key = %q, want clarify_request", result.ResponseKey)
	}
	if !result.LowConfidence {
		t.Fatalf("LowConfidence = false, want true")
	}
}

func TestDecisionServiceEmbeddingUnavailableAtRootKeepsClarifyFallback(t *testing.T) {
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
			{
				Key:            "ask_prices",
				Category:       "services",
				ResolutionType: "knowledge",
				ResponseKey:    "services_prices",
				Examples:       []string{"цены на услуги"},
			},
		},
	}, stubMatcher{result: MatchResult{
		IntentKey:      "ask_prices",
		Confidence:     0.88,
		LowConfidence:  true,
		FallbackReason: "embedding_unavailable",
		Candidates: []Candidate{
			{
				IntentKey:  "ask_prices",
				Confidence: 0.88,
				Source:     CandidateSourceLexicalFuzzy,
				Metadata: map[string]any{
					"reason": "embedding_unavailable",
				},
			},
		},
	}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Decide(
		context.Background(),
		session.Session{Mode: session.ModeStandard},
		nil,
		"совершенно непонятная просьба про космический тариф",
	)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}

	if result.ResponseKey != "clarify_request" {
		t.Fatalf("response_key = %q, want clarify_request when embedding is unavailable", result.ResponseKey)
	}
	if !result.LowConfidence {
		t.Fatalf("LowConfidence = false, want true when embedding is unavailable")
	}
	if result.FallbackReason != "embedding_unavailable" {
		t.Fatalf("FallbackReason = %q, want embedding_unavailable", result.FallbackReason)
	}
	if len(result.Candidates) == 0 {
		t.Fatalf("candidates = %#v, want lexical evidence preserved", result.Candidates)
	}
}

func TestDecisionServiceContextualFollowUpCarriesCandidateEvidence(t *testing.T) {
	t.Parallel()

	service, err := NewService(&apppresenter.IntentCatalog{
		Intents: []apppresenter.IntentDefinition{
			{
				Key:            "ask_payment_status",
				Category:       "payment",
				ResolutionType: "business_lookup",
				ResponseKey:    "payment_request_id",
				Action:         "find_payment",
				Examples:       []string{"статус платежа"},
			},
		},
	}, stubMatcher{result: MatchResult{
		IntentKey:      "",
		Confidence:     0.24,
		LowConfidence:  true,
		FallbackReason: defaultLowConfidence,
	}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Decide(
		context.Background(),
		session.Session{Mode: session.ModeStandard, ActiveTopic: "payment"},
		nil,
		"а статус?",
	)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}

	if result.Intent != "ask_payment_status" {
		t.Fatalf("intent = %q, want ask_payment_status", result.Intent)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates = %#v, want one contextual candidate", result.Candidates)
	}
	if result.Candidates[0].Source != CandidateSourceContextualRule {
		t.Fatalf("candidate source = %q, want contextual_rule", result.Candidates[0].Source)
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

func TestDecisionServiceClarifiesMixedWorkspaceAndServicesPriceQuery(t *testing.T) {
	t.Parallel()

	service, err := NewService(&apppresenter.IntentCatalog{
		Intents: []apppresenter.IntentDefinition{
			{
				Key:            "ask_workspace_prices",
				Category:       "workspace",
				ResolutionType: "knowledge",
				ResponseKey:    "workspace_types_prices",
				Examples:       []string{"сколько стоит рабочее место"},
			},
			{
				Key:            "ask_prices",
				Category:       "services",
				ResolutionType: "knowledge",
				ResponseKey:    "services_prices",
				Examples:       []string{"сколько стоят услуги"},
			},
		},
	}, stubMatcher{result: MatchResult{
		IntentKey:  "ask_workspace_prices",
		Confidence: 0.9176202668015928,
		Candidates: []Candidate{
			{
				IntentKey:  "ask_workspace_prices",
				Confidence: 0.9176202668015928,
				Source:     CandidateSourceIntentExample,
				Metadata:   map[string]any{"category": "workspace"},
			},
			{
				IntentKey:  "ask_prices",
				Confidence: 0.4295671042499613,
				Source:     CandidateSourceIntentExample,
				Metadata:   map[string]any{"category": "services"},
			},
		},
	}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Decide(context.Background(), session.Session{}, nil, "сколько стоит место и услуга")
	if err != nil {
		t.Fatalf("decide: %v", err)
	}

	if result.Intent != "unknown" || result.ResponseKey != "clarify_request" {
		t.Fatalf("result = %#v, want clarify for mixed-domain price query", result)
	}
	if !result.LowConfidence {
		t.Fatalf("LowConfidence = false, want true")
	}
}

func TestDecisionServiceClarifiesAmbiguousCodeQueryAcrossAccountAndTech(t *testing.T) {
	t.Parallel()

	service, err := NewService(&apppresenter.IntentCatalog{
		Intents: []apppresenter.IntentDefinition{
			{
				Key:            "code_not_received",
				Category:       "tech_issue",
				ResolutionType: "knowledge",
				ResponseKey:    "tech_code_not_received",
				Examples:       []string{"не приходит код для входа"},
			},
			{
				Key:            "account_code_not_received",
				Category:       "account",
				ResolutionType: "knowledge",
				ResponseKey:    "account_code_not_received",
				Examples:       []string{"не приходит код для аккаунта"},
			},
		},
	}, stubMatcher{result: MatchResult{
		IntentKey:  "code_not_received",
		Confidence: 1,
		Candidates: []Candidate{
			{
				IntentKey:  "code_not_received",
				Confidence: 1,
				Source:     CandidateSourceLexicalFuzzy,
				Metadata:   map[string]any{"category": "tech_issue"},
			},
			{
				IntentKey:  "account_code_not_received",
				Confidence: 0.8962617737104826,
				Source:     CandidateSourceLexicalFuzzy,
				Metadata:   map[string]any{"category": "account"},
			},
		},
	}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Decide(context.Background(), session.Session{}, nil, "код не приходит")
	if err != nil {
		t.Fatalf("decide: %v", err)
	}

	if result.Intent != "unknown" || result.ResponseKey != "clarify_request" {
		t.Fatalf("result = %#v, want clarify for ambiguous code query", result)
	}
	if !result.LowConfidence {
		t.Fatalf("LowConfidence = false, want true")
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

func TestDecisionServiceUsesIdentifierOnlyFollowUpForPendingBookingLookup(t *testing.T) {
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
	}, stubMatcher{result: MatchResult{}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Decide(context.Background(), session.Session{
		State:       state.StateWaitingForIdentifier,
		ActiveTopic: "booking",
		LastIntent:  "ask_booking_status",
	}, nil, "BRG-482910")
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if result.Intent != "ask_booking_status" {
		t.Fatalf("intent = %q, want ask_booking_status", result.Intent)
	}
	if got := result.ActionContext["provided_identifier"]; got != "BRG-482910" {
		t.Fatalf("provided_identifier = %#v, want BRG-482910", got)
	}
}

func TestDecisionServiceUsesContextualWorkspacePriceFollowUp(t *testing.T) {
	t.Parallel()

	service, err := NewService(&apppresenter.IntentCatalog{
		Intents: []apppresenter.IntentDefinition{
			{
				Key:            "ask_workspace_prices",
				Category:       "workspace",
				ResolutionType: "knowledge",
				ResponseKey:    "workspace_types_prices",
				Examples:       []string{"цены на коворкинг"},
			},
		},
	}, stubMatcher{result: MatchResult{
		IntentKey:      "",
		Confidence:     0.12,
		LowConfidence:  true,
		FallbackReason: defaultLowConfidence,
	}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Decide(context.Background(), session.Session{
		ActiveTopic: "workspace",
		State:       state.StateWorkspace,
	}, nil, "а сколько стоит")
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if result.Intent != "ask_workspace_prices" {
		t.Fatalf("intent = %q, want ask_workspace_prices", result.Intent)
	}
	if result.ResponseKey != "workspace_types_prices" {
		t.Fatalf("response_key = %q, want workspace_types_prices", result.ResponseKey)
	}
}

func TestDecisionServiceUsesContextualPaymentStatusFollowUp(t *testing.T) {
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
				Examples:            []string{"статус платежа"},
			},
		},
	}, stubMatcher{result: MatchResult{
		IntentKey:      "",
		Confidence:     0.2,
		LowConfidence:  true,
		FallbackReason: defaultLowConfidence,
	}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Decide(context.Background(), session.Session{
		ActiveTopic: "payment",
		State:       state.StatePayment,
	}, nil, "а статус")
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if result.Intent != "ask_payment_status" {
		t.Fatalf("intent = %q, want ask_payment_status", result.Intent)
	}
	if result.State != state.StateWaitingForIdentifier {
		t.Fatalf("state = %q, want waiting_for_identifier", result.State)
	}
}

func TestDecisionServiceNegativePendingFollowUpReturnsMenu(t *testing.T) {
	t.Parallel()

	service, err := NewService(&apppresenter.IntentCatalog{
		Intents: []apppresenter.IntentDefinition{
			{
				Key:            "return_to_menu",
				Category:       "system",
				ResolutionType: "static_response",
				ResponseKey:    "main_menu",
				Examples:       []string{"главное меню"},
			},
		},
	}, stubMatcher{result: MatchResult{
		IntentKey:      "",
		Confidence:     0.1,
		LowConfidence:  true,
		FallbackReason: defaultLowConfidence,
	}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Decide(context.Background(), session.Session{
		State:       state.StateWaitingForIdentifier,
		ActiveTopic: "booking",
		LastIntent:  "ask_booking_status",
	}, nil, "нет")
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if result.Intent != "return_to_menu" {
		t.Fatalf("intent = %q, want return_to_menu", result.Intent)
	}
	if result.ResponseKey != "main_menu" {
		t.Fatalf("response_key = %q, want main_menu", result.ResponseKey)
	}
}

func TestDecisionServiceAffirmativePendingFollowUpRepeatsLookupPrompt(t *testing.T) {
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
				Examples:            []string{"статус платежа"},
			},
		},
	}, stubMatcher{result: MatchResult{
		IntentKey:      "",
		Confidence:     0.1,
		LowConfidence:  true,
		FallbackReason: defaultLowConfidence,
	}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Decide(context.Background(), session.Session{
		State:       state.StateWaitingForIdentifier,
		ActiveTopic: "payment",
		LastIntent:  "ask_payment_status",
	}, nil, "да")
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if result.Intent != "ask_payment_status" {
		t.Fatalf("intent = %q, want ask_payment_status", result.Intent)
	}
	if result.State != state.StateWaitingForIdentifier {
		t.Fatalf("state = %q, want waiting_for_identifier", result.State)
	}
}

func TestDecisionServiceKeepsGlobalTopicSwitchWhenMatcherIsConfident(t *testing.T) {
	t.Parallel()

	service, err := NewService(&apppresenter.IntentCatalog{
		Intents: []apppresenter.IntentDefinition{
			{
				Key:            "payment_not_passed",
				Category:       "payment",
				ResolutionType: "knowledge",
				ResponseKey:    "payment_failed",
				Examples:       []string{"оплата не прошла"},
			},
			{
				Key:            "ask_workspace_prices",
				Category:       "workspace",
				ResolutionType: "knowledge",
				ResponseKey:    "workspace_types_prices",
				Examples:       []string{"цены на коворкинг"},
			},
		},
	}, stubMatcher{result: MatchResult{
		IntentKey:  "payment_not_passed",
		Confidence: 0.91,
		Candidates: []Candidate{
			{IntentKey: "payment_not_passed", Confidence: 0.91, Metadata: map[string]any{"category": "payment"}},
		},
	}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Decide(context.Background(), session.Session{
		ActiveTopic: "workspace",
		State:       state.StateWorkspace,
	}, nil, "оплата не прошла")
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if result.Intent != "payment_not_passed" {
		t.Fatalf("intent = %q, want payment_not_passed", result.Intent)
	}
	if result.Topic != "payment" {
		t.Fatalf("topic = %q, want payment", result.Topic)
	}
}

func TestDecisionServiceEnrichesKnowledgeIntentWithKnowledgeChunkCandidate(t *testing.T) {
	t.Parallel()

	searcher := &stubKnowledgeSearcher{
		candidate: &Candidate{
			IntentKey:  "ask_workspace_rules",
			Confidence: 0.88,
			Source:     CandidateSourceKnowledgeChunk,
			Text:       "В коворкинге нельзя шуметь после 22:00",
			Metadata: map[string]any{
				"article_key": "workspace.rules",
				"chunk_index": 0,
			},
		},
	}
	service, err := NewService(&apppresenter.IntentCatalog{
		Intents: []apppresenter.IntentDefinition{
			{
				Key:            "ask_workspace_rules",
				Category:       "workspace",
				ResolutionType: "knowledge",
				ResponseKey:    "workspace_rental_rules",
				KnowledgeKey:   "workspace.rules",
				Examples:       []string{"правила аренды"},
			},
		},
	}, stubMatcher{result: MatchResult{
		IntentKey:  "ask_workspace_rules",
		Confidence: 0.9,
		Candidates: []Candidate{
			{IntentKey: "ask_workspace_rules", Confidence: 0.9, Source: CandidateSourceIntentExample},
		},
	}}, logger.Noop())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	service.SetKnowledgeSearcher(searcher)

	result, err := service.Decide(context.Background(), session.Session{}, nil, "можно шуметь в коворкинге")
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if result.Intent != "ask_workspace_rules" {
		t.Fatalf("intent = %q, want ask_workspace_rules", result.Intent)
	}
	if searcher.calls != 1 {
		t.Fatalf("knowledge search calls = %d, want 1", searcher.calls)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("candidates = %#v, want matcher candidate plus knowledge chunk", result.Candidates)
	}
	if result.Candidates[1].Source != CandidateSourceKnowledgeChunk {
		t.Fatalf("knowledge source = %q, want knowledge_chunk", result.Candidates[1].Source)
	}
	if got := result.Candidates[1].Metadata["article_key"]; got != "workspace.rules" {
		t.Fatalf("article_key = %#v, want workspace.rules", got)
	}
}
