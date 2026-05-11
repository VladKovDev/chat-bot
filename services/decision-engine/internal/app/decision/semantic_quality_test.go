package decision

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
)

var minimumSemanticQualityRates = map[string]float64{
	"top1_intent_accuracy":          0.95,
	"top3_contains_expected":        0.95,
	"clarify_expectation_accuracy":  0.95,
	"operator_expectation_accuracy": 0.99,
}

var minimumSemanticCategoryTop1Rates = map[string]float64{
	"account":    0.80,
	"booking":    0.90,
	"complaint":  0.90,
	"fallback":   0.70,
	"operator":   0.80,
	"other":      0.80,
	"payment":    0.90,
	"services":   0.90,
	"system":     0.90,
	"tech_issue": 0.80,
	"workspace":  0.90,
}

func TestSemanticGoldCorpusQualityBaseline(t *testing.T) {
	t.Parallel()

	report, err := EvaluateSemanticCorpus(context.Background(), SemanticEvaluationConfig{
		CorpusPath:  filepath.Join("testdata", "semantic_gold_corpus.json"),
		CatalogPath: filepath.Join("..", "..", "..", ".."),
	})
	if err != nil {
		t.Fatalf("evaluate semantic corpus: %v", err)
	}

	summary := report.Summary
	if summary.Total < 150 {
		t.Fatalf("semantic corpus total = %d, want at least 150", summary.Total)
	}
	if len(summary.ByCategory) < 10 {
		t.Fatalf("semantic corpus category coverage = %d, want at least 10", len(summary.ByCategory))
	}

	assertBaseline(t, "total", summary.Total, 207)
	assertBaseline(t, "top1", summary.Top1IntentCorrect, 204)
	assertBaseline(t, "top3", summary.Top3ContainsExpected, 204)
	assertBaseline(t, "clarify", summary.ClarifyExpectationCorrect, 204)
	assertBaseline(t, "operator", summary.OperatorExpectationCorrect, 207)
}

func TestSemanticGoldCorpusFailureDiagnostics(t *testing.T) {
	t.Parallel()

	report, err := EvaluateSemanticCorpus(context.Background(), SemanticEvaluationConfig{
		CorpusPath:  filepath.Join("testdata", "semantic_gold_corpus.json"),
		CatalogPath: filepath.Join("..", "..", "..", ".."),
	})
	if err != nil {
		t.Fatalf("evaluate semantic corpus: %v", err)
	}
	if len(report.Summary.Failures) == 0 {
		t.Fatalf("expected baseline to expose failure diagnostics before semantic tuning")
	}
	failure := report.Summary.Failures[0]
	if failure.ID == "" || failure.Text == "" || failure.Reason == "" {
		t.Fatalf("failure diagnostic missing case identity/reason: %#v", failure)
	}
	if failure.ExpectedIntent == "" && !failure.ExpectedClarify && !failure.ExpectedOperator {
		t.Fatalf("failure diagnostic missing expectation: %#v", failure)
	}
}

func TestSemanticGoldCorpusMeetsReleaseThresholds(t *testing.T) {
	t.Parallel()

	report, err := EvaluateSemanticCorpus(context.Background(), SemanticEvaluationConfig{
		CorpusPath:  filepath.Join("testdata", "semantic_gold_corpus.json"),
		CatalogPath: filepath.Join("..", "..", "..", ".."),
	})
	if err != nil {
		t.Fatalf("evaluate semantic corpus: %v", err)
	}

	rates := report.Summary.Rates()
	for key, minimum := range minimumSemanticQualityRates {
		got := rates[key]
		if got < minimum {
			t.Fatalf("%s = %.4f, want >= %.4f", key, got, minimum)
		}
	}

	for category, minimum := range minimumSemanticCategoryTop1Rates {
		metrics, ok := report.Summary.ByCategory[category]
		if !ok {
			t.Fatalf("missing category metrics for %s", category)
		}
		got := float64(metrics.Top1IntentCorrect) / float64(metrics.Total)
		if got < minimum {
			t.Fatalf("%s top1 = %.4f, want >= %.4f", category, got, minimum)
		}
	}
}

func TestIsClarifyDecisionDoesNotTreatOperatorHandoffAsClarification(t *testing.T) {
	t.Parallel()

	result := Result{
		Intent:         "unknown",
		State:          state.StateEscalatedToOperator,
		ResponseKey:    "operator_handoff_requested",
		LowConfidence:  true,
		Event:          session.EventRequestOperator,
		FallbackReason: "low_confidence_repeated",
	}

	if isClarifyDecision(result) {
		t.Fatalf("operator handoff should not count as clarify: %#v", result)
	}
}

func assertBaseline(t *testing.T, name string, got, want int) {
	t.Helper()
	if got == want {
		return
	}
	t.Fatalf("%s baseline = %d, want %d", name, got, want)
}
