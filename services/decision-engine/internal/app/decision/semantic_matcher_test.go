package decision

import (
	"context"
	"errors"
	"testing"

	apppresenter "github.com/VladKovDev/chat-bot/internal/app/presenter"
)

type fakeEmbedder struct {
	embedding []float64
	err       error
	calls     int
}

func (e *fakeEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	e.calls++
	return append([]float64(nil), e.embedding...), e.err
}

type fakeIntentSearch struct {
	rows  []IntentSearchResult
	calls int
}

func (s *fakeIntentSearch) SearchIntentExamples(
	_ context.Context,
	_ []float64,
	_ string,
	_ int,
) ([]IntentSearchResult, error) {
	s.calls++
	return append([]IntentSearchResult(nil), s.rows...), nil
}

func TestSemanticMatcherSelectsConfidentIntentAndCandidates(t *testing.T) {
	t.Parallel()

	embedder := &fakeEmbedder{embedding: []float64{1, 0, 0}}
	search := &fakeIntentSearch{rows: []IntentSearchResult{
		{
			IntentID:       "intent-payment",
			IntentKey:      "ask_payment_status",
			Category:       "payment",
			ResponseKey:    "payment_request_identifier",
			Text:           "проверь оплату",
			NormalizedText: "проверь оплату",
			Locale:         "ru",
			Weight:         1,
			Confidence:     0.92,
		},
		{
			IntentID:       "intent-booking",
			IntentKey:      "ask_booking_status",
			Category:       "booking",
			ResponseKey:    "booking_request_identifier",
			Text:           "проверь запись",
			NormalizedText: "проверь запись",
			Locale:         "ru",
			Weight:         1,
			Confidence:     0.72,
		},
	}}
	matcher, err := NewSemanticIntentMatcher(embedder, search, SemanticMatcherConfig{TopK: 3})
	if err != nil {
		t.Fatalf("new semantic matcher: %v", err)
	}

	match, err := matcher.Match(context.Background(), "Где моя оплата?", nil)
	if err != nil {
		t.Fatalf("match: %v", err)
	}

	if match.IntentKey != "ask_payment_status" || match.Confidence != 0.92 {
		t.Fatalf("match = %#v, want ask_payment_status confidence 0.92", match)
	}
	if match.LowConfidence {
		t.Fatalf("LowConfidence = true, want false")
	}
	if match.AmbiguityDelta < DefaultAmbiguityDelta {
		t.Fatalf("ambiguity delta = %.2f, want >= %.2f", match.AmbiguityDelta, DefaultAmbiguityDelta)
	}
	if len(match.Candidates) != 2 {
		t.Fatalf("candidate count = %d, want 2", len(match.Candidates))
	}
	if match.Candidates[0].Source != CandidateSourceIntentExample || match.Candidates[0].Text == "" {
		t.Fatalf("candidate metadata missing: %#v", match.Candidates[0])
	}
}

func TestSemanticMatcherMarksLowConfidenceAndAmbiguousMatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		rows           []IntentSearchResult
		wantReason     string
		wantConfidence float64
	}{
		{
			name: "low confidence",
			rows: []IntentSearchResult{
				{IntentKey: "ask_payment_status", Weight: 1, Confidence: 0.77},
			},
			wantReason:     defaultLowConfidence,
			wantConfidence: 0.77,
		},
		{
			name: "ambiguous",
			rows: []IntentSearchResult{
				{IntentKey: "ask_payment_status", Weight: 1, Confidence: 0.88},
				{IntentKey: "ask_booking_status", Weight: 1, Confidence: 0.83},
			},
			wantReason:     defaultAmbiguousMatch,
			wantConfidence: 0.88,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			matcher, err := NewSemanticIntentMatcher(
				&fakeEmbedder{embedding: []float64{1, 0, 0}},
				&fakeIntentSearch{rows: tt.rows},
				SemanticMatcherConfig{},
			)
			if err != nil {
				t.Fatalf("new semantic matcher: %v", err)
			}

			match, err := matcher.Match(context.Background(), "неочевидный запрос", nil)
			if err != nil {
				t.Fatalf("match: %v", err)
			}
			if !match.LowConfidence || match.FallbackReason != tt.wantReason {
				t.Fatalf("low confidence = %t reason = %q, want true/%q", match.LowConfidence, match.FallbackReason, tt.wantReason)
			}
			if match.Confidence != tt.wantConfidence {
				t.Fatalf("confidence = %.2f, want %.2f", match.Confidence, tt.wantConfidence)
			}
		})
	}
}

func TestSemanticMatcherExactCommandsBypassEmbeddingOutage(t *testing.T) {
	t.Parallel()

	embedder := &fakeEmbedder{err: errors.New("nlp unavailable")}
	search := &fakeIntentSearch{}
	matcher, err := NewSemanticIntentMatcher(embedder, search, SemanticMatcherConfig{})
	if err != nil {
		t.Fatalf("new semantic matcher: %v", err)
	}

	match, err := matcher.Match(context.Background(), " Главное меню ", []apppresenter.IntentDefinition{
		{
			Key:      "return_to_menu",
			Category: "system",
			Examples: []string{"главное меню"},
		},
	})
	if err != nil {
		t.Fatalf("match: %v", err)
	}
	if match.IntentKey != "return_to_menu" || match.Confidence != 1 {
		t.Fatalf("match = %#v, want exact return_to_menu", match)
	}
	if embedder.calls != 0 || search.calls != 0 {
		t.Fatalf("embed/search calls = %d/%d, want 0/0 for exact command", embedder.calls, search.calls)
	}
	if match.Candidates[0].Source != CandidateSourceExactCommand {
		t.Fatalf("candidate source = %q, want exact_command", match.Candidates[0].Source)
	}
}

func TestSemanticMatcherEmbeddingOutageReturnsLowConfidenceFallback(t *testing.T) {
	t.Parallel()

	matcher, err := NewSemanticIntentMatcher(
		&fakeEmbedder{err: errors.New("nlp unavailable")},
		&fakeIntentSearch{},
		SemanticMatcherConfig{},
	)
	if err != nil {
		t.Fatalf("new semantic matcher: %v", err)
	}

	match, err := matcher.Match(context.Background(), "что-то сложное", nil)
	if err != nil {
		t.Fatalf("match: %v", err)
	}
	if !match.LowConfidence || match.FallbackReason != "embedding_unavailable" {
		t.Fatalf("match = %#v, want embedding outage low confidence", match)
	}
	if len(match.Candidates) != 1 || match.Candidates[0].Source != CandidateSourceFallback {
		t.Fatalf("fallback candidates = %#v, want one fallback candidate", match.Candidates)
	}
}
