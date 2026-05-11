package decision

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	apppresenter "github.com/VladKovDev/chat-bot/internal/app/presenter"
)

const (
	CandidateSourceIntentExample = "intent_example"
	CandidateSourceExactCommand  = "exact_command"
	CandidateSourceLexicalFuzzy  = "lexical_fuzzy"
	CandidateSourceFallback      = "fallback"
)

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

type IntentSearchRepository interface {
	SearchIntentExamples(ctx context.Context, embedding []float64, locale string, limit int) ([]IntentSearchResult, error)
}

type IntentSearchResult struct {
	IntentID       string
	IntentKey      string
	Category       string
	ResponseKey    string
	Text           string
	NormalizedText string
	Locale         string
	Weight         float64
	Confidence     float64
}

type SemanticMatcherConfig struct {
	TopK   int
	Locale string
}

type SemanticIntentMatcher struct {
	embedder Embedder
	search   IntentSearchRepository
	topK     int
	locale   string
}

func NewSemanticIntentMatcher(
	embedder Embedder,
	search IntentSearchRepository,
	cfg SemanticMatcherConfig,
) (*SemanticIntentMatcher, error) {
	if embedder == nil {
		return nil, fmt.Errorf("semantic matcher embedder is required")
	}
	if search == nil {
		return nil, fmt.Errorf("semantic matcher search repository is required")
	}
	topK := cfg.TopK
	if topK <= 0 {
		topK = 3
	}
	locale := strings.TrimSpace(cfg.Locale)
	if locale == "" {
		locale = "ru"
	}
	return &SemanticIntentMatcher{
		embedder: embedder,
		search:   search,
		topK:     topK,
		locale:   locale,
	}, nil
}

func (m *SemanticIntentMatcher) Match(
	ctx context.Context,
	text string,
	intents []apppresenter.IntentDefinition,
) (MatchResult, error) {
	if exact := exactCommandMatch(text, intents); exact.IntentKey != "" {
		return exact, nil
	}

	lexicalCandidates := lexicalIntentCandidates(text, intents, m.topK)

	embedding, err := m.embedder.Embed(ctx, text)
	if err != nil {
		if len(lexicalCandidates) > 0 {
			return rankCandidates(lexicalCandidates), nil
		}
		return MatchResult{
			LowConfidence:  true,
			FallbackReason: "embedding_unavailable",
			Candidates: []Candidate{
				{
					IntentKey:  "unknown",
					Confidence: 0,
					Source:     CandidateSourceFallback,
					Metadata: map[string]any{
						"reason": "embedding_unavailable",
					},
				},
			},
		}, nil
	}

	rows, err := m.search.SearchIntentExamples(ctx, embedding, m.locale, m.topK*3)
	if err != nil {
		if len(lexicalCandidates) > 0 {
			return rankCandidates(lexicalCandidates), nil
		}
		return MatchResult{}, fmt.Errorf("search intent examples: %w", err)
	}

	allowed := allowedIntentKeys(intents)
	semanticCandidates := uniqueIntentCandidates(rows, allowed, m.topK)
	candidates := mergeCandidates(semanticCandidates, lexicalCandidates, m.topK)
	if len(candidates) == 0 {
		return MatchResult{
			LowConfidence:  true,
			FallbackReason: defaultNoSemanticIntent,
		}, nil
	}

	match := MatchResult{
		IntentKey:      candidates[0].IntentKey,
		Confidence:     candidates[0].Confidence,
		Candidates:     candidates,
		AmbiguityDelta: 1,
	}
	if len(candidates) > 1 {
		match.AmbiguityDelta = candidates[0].Confidence - candidates[1].Confidence
	}
	if match.Confidence < DefaultMatchThreshold {
		match.LowConfidence = true
		match.FallbackReason = defaultLowConfidence
	} else if len(candidates) > 1 && match.AmbiguityDelta < DefaultAmbiguityDelta {
		match.LowConfidence = true
		match.FallbackReason = defaultAmbiguousMatch
	}
	return match, nil
}

func exactCommandMatch(text string, intents []apppresenter.IntentDefinition) MatchResult {
	normalizedQuery := normalizeText(text)
	if normalizedQuery == "" {
		return MatchResult{}
	}
	for _, intentDefinition := range intents {
		for _, example := range intentDefinition.Examples {
			if normalizedQuery != normalizeText(example) {
				continue
			}
			candidate := Candidate{
				IntentKey:  intentDefinition.Key,
				Confidence: 1,
				Source:     CandidateSourceExactCommand,
				Text:       example,
			}
			return MatchResult{
				IntentKey:      intentDefinition.Key,
				Confidence:     1,
				AmbiguityDelta: 1,
				Candidates:     []Candidate{candidate},
			}
		}
	}
	return MatchResult{}
}

func uniqueIntentCandidates(rows []IntentSearchResult, allowed map[string]struct{}, limit int) []Candidate {
	if limit <= 0 || len(rows) == 0 {
		return nil
	}

	byIntent := make(map[string]Candidate, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.IntentKey) == "" {
			continue
		}
		if len(allowed) > 0 {
			if _, ok := allowed[row.IntentKey]; !ok {
				continue
			}
		}
		confidence := row.Confidence
		if row.Weight > 0 {
			confidence *= row.Weight
		}
		if confidence > 1 {
			confidence = 1
		}
		if confidence < 0 {
			confidence = 0
		}
		candidate := Candidate{
			IntentID:   row.IntentID,
			IntentKey:  row.IntentKey,
			Confidence: confidence,
			Source:     CandidateSourceIntentExample,
			Text:       row.Text,
			Metadata: map[string]any{
				"category":        row.Category,
				"response_key":    row.ResponseKey,
				"normalized_text": row.NormalizedText,
				"locale":          row.Locale,
				"weight":          row.Weight,
			},
		}
		current, exists := byIntent[row.IntentKey]
		if !exists || candidate.Confidence > current.Confidence {
			byIntent[row.IntentKey] = candidate
		}
	}

	candidates := make([]Candidate, 0, len(byIntent))
	for _, candidate := range byIntent {
		candidates = append(candidates, candidate)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Confidence == candidates[j].Confidence {
			return candidates[i].IntentKey < candidates[j].IntentKey
		}
		return candidates[i].Confidence > candidates[j].Confidence
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

func allowedIntentKeys(intents []apppresenter.IntentDefinition) map[string]struct{} {
	if len(intents) == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(intents))
	for _, intentDefinition := range intents {
		if strings.TrimSpace(intentDefinition.Key) == "" {
			continue
		}
		allowed[intentDefinition.Key] = struct{}{}
	}
	return allowed
}

func mergeCandidates(semanticCandidates, lexicalCandidates []Candidate, limit int) []Candidate {
	if len(semanticCandidates) == 0 {
		return lexicalCandidates
	}
	if len(lexicalCandidates) == 0 {
		return semanticCandidates
	}

	merged := make(map[string]Candidate, len(semanticCandidates)+len(lexicalCandidates))
	for _, candidate := range semanticCandidates {
		merged[candidate.IntentKey] = candidate
	}
	for _, lexical := range lexicalCandidates {
		current, ok := merged[lexical.IntentKey]
		if !ok {
			merged[lexical.IntentKey] = lexical
			continue
		}

		combined := current
		combined.Confidence = normalizeConfidence(math.Max(current.Confidence, lexical.Confidence) + 0.08)
		if lexical.Confidence > current.Confidence {
			combined.Source = lexical.Source
			combined.Text = lexical.Text
		}
		if combined.Metadata == nil {
			combined.Metadata = map[string]any{}
		}
		combined.Metadata["matched_sources"] = []string{current.Source, lexical.Source}
		merged[lexical.IntentKey] = combined
	}

	candidates := make([]Candidate, 0, len(merged))
	for _, candidate := range merged {
		candidates = append(candidates, candidate)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Confidence == candidates[j].Confidence {
			return candidates[i].IntentKey < candidates[j].IntentKey
		}
		return candidates[i].Confidence > candidates[j].Confidence
	})
	if limit > 0 && len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

func rankCandidates(candidates []Candidate) MatchResult {
	match := MatchResult{
		IntentKey:      candidates[0].IntentKey,
		Confidence:     candidates[0].Confidence,
		Candidates:     append([]Candidate(nil), candidates...),
		AmbiguityDelta: 1,
	}
	if len(candidates) > 1 {
		match.AmbiguityDelta = candidates[0].Confidence - candidates[1].Confidence
	}
	if match.Confidence < DefaultMatchThreshold {
		match.LowConfidence = true
		match.FallbackReason = defaultLowConfidence
	} else if len(candidates) > 1 && match.AmbiguityDelta < DefaultAmbiguityDelta {
		match.LowConfidence = true
		match.FallbackReason = defaultAmbiguousMatch
	}
	return match
}
