package decision

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	apppresenter "github.com/VladKovDev/chat-bot/internal/app/presenter"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

type SemanticCorpus struct {
	Version int                   `json:"version"`
	Groups  []SemanticCorpusGroup `json:"groups"`
	Cases   []SemanticCorpusCase  `json:"cases"`
}

type SemanticCorpusGroup struct {
	ID               string   `json:"id"`
	Category         string   `json:"category"`
	ExpectedIntent   string   `json:"expected_intent"`
	ExpectedTop3     bool     `json:"expected_top3"`
	ExpectedClarify  bool     `json:"expected_clarify"`
	ExpectedOperator bool     `json:"expected_operator"`
	ActiveTopic      string   `json:"active_topic,omitempty"`
	FallbackCount    int      `json:"fallback_count,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	Utterances       []string `json:"utterances"`
}

type SemanticCorpusCase struct {
	ID               string   `json:"id"`
	Text             string   `json:"text"`
	Category         string   `json:"category"`
	ExpectedIntent   string   `json:"expected_intent"`
	ExpectedTop3     bool     `json:"expected_top3"`
	ExpectedClarify  bool     `json:"expected_clarify"`
	ExpectedOperator bool     `json:"expected_operator"`
	ActiveTopic      string   `json:"active_topic,omitempty"`
	FallbackCount    int      `json:"fallback_count,omitempty"`
	Tags             []string `json:"tags,omitempty"`
}

type SemanticEvaluationConfig struct {
	CorpusPath  string
	CatalogPath string
}

type SemanticEvaluationResult struct {
	Total                      int                                `json:"total"`
	Top1IntentCorrect          int                                `json:"top1_intent_correct"`
	Top3ContainsExpected       int                                `json:"top3_contains_expected"`
	ClarifyExpectationCorrect  int                                `json:"clarify_expectation_correct"`
	OperatorExpectationCorrect int                                `json:"operator_expectation_correct"`
	ContextSensitiveCorrect    int                                `json:"context_sensitive_correct"`
	ByCategory                 map[string]SemanticCategoryMetrics `json:"by_category"`
	Failures                   []SemanticCaseFailure              `json:"failures"`
}

type SemanticCategoryMetrics struct {
	Total                int `json:"total"`
	Top1IntentCorrect    int `json:"top1_intent_correct"`
	Top3ContainsExpected int `json:"top3_contains_expected"`
}

type SemanticCaseFailure struct {
	ID               string      `json:"id"`
	Text             string      `json:"text"`
	Category         string      `json:"category"`
	ExpectedIntent   string      `json:"expected_intent,omitempty"`
	ActualIntent     string      `json:"actual_intent"`
	ExpectedClarify  bool        `json:"expected_clarify,omitempty"`
	ActualClarify    bool        `json:"actual_clarify"`
	ExpectedOperator bool        `json:"expected_operator,omitempty"`
	ActualOperator   bool        `json:"actual_operator"`
	ResponseKey      string      `json:"response_key"`
	Confidence       *float64    `json:"confidence,omitempty"`
	Candidates       []Candidate `json:"candidates,omitempty"`
	Reason           string      `json:"reason"`
}

type SemanticEvaluationReport struct {
	Summary SemanticEvaluationResult `json:"summary"`
}

func LoadSemanticCorpus(path string) ([]SemanticCorpusCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read semantic corpus: %w", err)
	}

	var corpus SemanticCorpus
	if err := json.Unmarshal(data, &corpus); err != nil {
		return nil, fmt.Errorf("parse semantic corpus: %w", err)
	}

	cases := make([]SemanticCorpusCase, 0, len(corpus.Cases))
	seen := make(map[string]struct{})
	for _, group := range corpus.Groups {
		for index, utterance := range group.Utterances {
			id := fmt.Sprintf("%s.%03d", group.ID, index+1)
			if _, exists := seen[id]; exists {
				return nil, fmt.Errorf("duplicate semantic corpus case id %q", id)
			}
			seen[id] = struct{}{}
			cases = append(cases, SemanticCorpusCase{
				ID:               id,
				Text:             utterance,
				Category:         group.Category,
				ExpectedIntent:   group.ExpectedIntent,
				ExpectedTop3:     group.ExpectedTop3,
				ExpectedClarify:  group.ExpectedClarify,
				ExpectedOperator: group.ExpectedOperator,
				ActiveTopic:      group.ActiveTopic,
				FallbackCount:    group.FallbackCount,
				Tags:             append([]string(nil), group.Tags...),
			})
		}
	}
	for _, testCase := range corpus.Cases {
		if _, exists := seen[testCase.ID]; exists {
			return nil, fmt.Errorf("duplicate semantic corpus case id %q", testCase.ID)
		}
		seen[testCase.ID] = struct{}{}
		cases = append(cases, testCase)
	}

	for _, testCase := range cases {
		if strings.TrimSpace(testCase.ID) == "" {
			return nil, fmt.Errorf("semantic corpus case has empty id")
		}
		if strings.TrimSpace(testCase.Text) == "" {
			return nil, fmt.Errorf("semantic corpus case %q has empty text", testCase.ID)
		}
		if !testCase.ExpectedClarify && !testCase.ExpectedOperator && strings.TrimSpace(testCase.ExpectedIntent) == "" {
			return nil, fmt.Errorf("semantic corpus case %q has no expectation", testCase.ID)
		}
	}

	return cases, nil
}

func EvaluateSemanticCorpus(ctx context.Context, cfg SemanticEvaluationConfig) (SemanticEvaluationReport, error) {
	cases, err := LoadSemanticCorpus(cfg.CorpusPath)
	if err != nil {
		return SemanticEvaluationReport{}, err
	}
	catalog, err := apppresenter.LoadIntentCatalog(cfg.CatalogPath)
	if err != nil {
		return SemanticEvaluationReport{}, err
	}
	embedder := newDeterministicHashEmbedder(corpusHarnessEmbeddingDimension, "semantic-gate")
	searchRepo, err := newInMemoryIntentSearchRepository(ctx, embedder, catalog.Intents)
	if err != nil {
		return SemanticEvaluationReport{}, err
	}
	matcher, err := NewSemanticIntentMatcher(embedder, searchRepo, SemanticMatcherConfig{TopK: 3, Locale: "ru"})
	if err != nil {
		return SemanticEvaluationReport{}, err
	}
	service, err := NewService(catalog, matcher, logger.Noop())
	if err != nil {
		return SemanticEvaluationReport{}, err
	}

	result := SemanticEvaluationResult{
		ByCategory: make(map[string]SemanticCategoryMetrics),
	}

	for _, testCase := range cases {
		sess := session.Session{
			Mode:          session.ModeStandard,
			State:         state.StateWaitingForCategory,
			ActiveTopic:   testCase.ActiveTopic,
			FallbackCount: testCase.FallbackCount,
		}
		decision, err := service.Decide(ctx, sess, nil, testCase.Text)
		if err != nil {
			return SemanticEvaluationReport{}, fmt.Errorf("evaluate semantic case %s: %w", testCase.ID, err)
		}

		result.Total++
		category := strings.TrimSpace(testCase.Category)
		if category == "" {
			category = "uncategorized"
		}
		categoryMetrics := result.ByCategory[category]
		categoryMetrics.Total++

		top1OK := strings.TrimSpace(testCase.ExpectedIntent) != "" && decision.Intent == testCase.ExpectedIntent
		if top1OK {
			result.Top1IntentCorrect++
			categoryMetrics.Top1IntentCorrect++
		}
		top3OK := !testCase.ExpectedTop3 || candidateContainsIntent(decision.Candidates, testCase.ExpectedIntent)
		if top3OK && testCase.ExpectedTop3 {
			result.Top3ContainsExpected++
			categoryMetrics.Top3ContainsExpected++
		}
		clarifyOK := isClarifyDecision(decision) == testCase.ExpectedClarify
		if clarifyOK {
			result.ClarifyExpectationCorrect++
		}
		operatorOK := isOperatorDecision(decision) == testCase.ExpectedOperator
		if operatorOK {
			result.OperatorExpectationCorrect++
		}
		if testCase.ActiveTopic != "" && top1OK {
			result.ContextSensitiveCorrect++
		}

		result.ByCategory[category] = categoryMetrics
		if top1OK && top3OK && clarifyOK && operatorOK {
			continue
		}

		result.Failures = append(result.Failures, SemanticCaseFailure{
			ID:               testCase.ID,
			Text:             testCase.Text,
			Category:         category,
			ExpectedIntent:   testCase.ExpectedIntent,
			ActualIntent:     decision.Intent,
			ExpectedClarify:  testCase.ExpectedClarify,
			ActualClarify:    isClarifyDecision(decision),
			ExpectedOperator: testCase.ExpectedOperator,
			ActualOperator:   isOperatorDecision(decision),
			ResponseKey:      decision.ResponseKey,
			Confidence:       decision.Confidence,
			Candidates:       append([]Candidate(nil), decision.Candidates...),
			Reason:           semanticFailureReason(testCase, decision, top1OK, top3OK, clarifyOK, operatorOK),
		})
	}

	return SemanticEvaluationReport{Summary: result}, nil
}

func (r SemanticEvaluationResult) Rates() map[string]float64 {
	if r.Total == 0 {
		return map[string]float64{}
	}
	total := float64(r.Total)
	return map[string]float64{
		"top1_intent_accuracy":          float64(r.Top1IntentCorrect) / total,
		"top3_contains_expected":        float64(r.Top3ContainsExpected) / total,
		"clarify_expectation_accuracy":  float64(r.ClarifyExpectationCorrect) / total,
		"operator_expectation_accuracy": float64(r.OperatorExpectationCorrect) / total,
	}
}

func (r SemanticEvaluationResult) SortedCategories() []string {
	categories := make([]string, 0, len(r.ByCategory))
	for category := range r.ByCategory {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	return categories
}

func candidateContainsIntent(candidates []Candidate, intentKey string) bool {
	if strings.TrimSpace(intentKey) == "" {
		return true
	}
	for _, candidate := range candidates {
		if candidate.IntentKey == intentKey {
			return true
		}
	}
	return false
}

func isClarifyDecision(result Result) bool {
	if isOperatorDecision(result) {
		return false
	}
	return result.LowConfidence || result.ResponseKey == "clarify_request" || result.State == state.StateWaitingClarification
}

func isOperatorDecision(result Result) bool {
	return result.Event == session.EventRequestOperator || result.State == state.StateEscalatedToOperator
}

func semanticFailureReason(testCase SemanticCorpusCase, result Result, top1OK, top3OK, clarifyOK, operatorOK bool) string {
	reasons := make([]string, 0, 4)
	if !top1OK {
		reasons = append(reasons, fmt.Sprintf("top1 intent expected %q actual %q", testCase.ExpectedIntent, result.Intent))
	}
	if !top3OK {
		reasons = append(reasons, fmt.Sprintf("expected intent %q missing from top3 candidates", testCase.ExpectedIntent))
	}
	if !clarifyOK {
		reasons = append(reasons, fmt.Sprintf("clarify expected %t actual %t", testCase.ExpectedClarify, isClarifyDecision(result)))
	}
	if !operatorOK {
		reasons = append(reasons, fmt.Sprintf("operator expected %t actual %t", testCase.ExpectedOperator, isOperatorDecision(result)))
	}
	return strings.Join(reasons, "; ")
}
