package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/VladKovDev/chat-bot/internal/app/decision"
)

func main() {
	format := flag.String("format", "table", "output format: table or json")
	corpusPath := flag.String("corpus", filepath.Join("internal", "app", "decision", "testdata", "semantic_gold_corpus.json"), "semantic corpus path")
	catalogPath := flag.String("catalog", ".", "path used to locate seeds/intents.json")
	flag.Parse()

	report, err := decision.EvaluateSemanticCorpus(context.Background(), decision.SemanticEvaluationConfig{
		CorpusPath:  *corpusPath,
		CatalogPath: *catalogPath,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "semantic gate failed: %v\n", err)
		os.Exit(1)
	}

	switch *format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "encode semantic report: %v\n", err)
			os.Exit(1)
		}
	case "table":
		printTable(report.Summary)
	default:
		fmt.Fprintf(os.Stderr, "unsupported format %q\n", *format)
		os.Exit(1)
	}
}

func printTable(summary decision.SemanticEvaluationResult) {
	rates := summary.Rates()
	fmt.Printf("semantic corpus cases: %d\n", summary.Total)
	fmt.Printf("top1 intent accuracy: %.2f%% (%d/%d)\n", rates["top1_intent_accuracy"]*100, summary.Top1IntentCorrect, summary.Total)
	fmt.Printf("top3 contains expected: %.2f%% (%d/%d)\n", rates["top3_contains_expected"]*100, summary.Top3ContainsExpected, summary.Total)
	fmt.Printf("clarify expectation: %.2f%% (%d/%d)\n", rates["clarify_expectation_accuracy"]*100, summary.ClarifyExpectationCorrect, summary.Total)
	fmt.Printf("operator expectation: %.2f%% (%d/%d)\n", rates["operator_expectation_accuracy"]*100, summary.OperatorExpectationCorrect, summary.Total)
	fmt.Println()
	fmt.Println("category\ttotal\ttop1\ttop3")
	for _, category := range summary.SortedCategories() {
		metrics := summary.ByCategory[category]
		fmt.Printf("%s\t%d\t%d\t%d\n", category, metrics.Total, metrics.Top1IntentCorrect, metrics.Top3ContainsExpected)
	}
	if len(summary.Failures) == 0 {
		return
	}
	fmt.Println()
	fmt.Printf("failures: %d\n", len(summary.Failures))
	for index, failure := range summary.Failures {
		if index >= 10 {
			fmt.Printf("... %d more\n", len(summary.Failures)-index)
			break
		}
		fmt.Printf("- %s expected=%s actual=%s response=%s reason=%s\n", failure.ID, failure.ExpectedIntent, failure.ActualIntent, failure.ResponseKey, failure.Reason)
	}
}
