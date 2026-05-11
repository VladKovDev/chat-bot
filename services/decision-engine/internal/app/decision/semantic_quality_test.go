package decision

import (
	"context"
	"path/filepath"
	"testing"
)

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
	assertBaseline(t, "top1", summary.Top1IntentCorrect, 152)
	assertBaseline(t, "top3", summary.Top3ContainsExpected, 174)
	assertBaseline(t, "clarify", summary.ClarifyExpectationCorrect, 201)
	assertBaseline(t, "operator", summary.OperatorExpectationCorrect, 203)
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

func assertBaseline(t *testing.T, name string, got, want int) {
	t.Helper()
	if got == want {
		return
	}
	t.Fatalf("%s baseline = %d, want %d", name, got, want)
}
