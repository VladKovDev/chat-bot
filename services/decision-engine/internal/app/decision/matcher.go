package decision

import (
	"context"
	"sort"
	"strings"
	"unicode"

	apppresenter "github.com/VladKovDev/chat-bot/internal/app/presenter"
)

type CatalogMatcher struct{}

func NewCatalogMatcher() *CatalogMatcher {
	return &CatalogMatcher{}
}

func (m *CatalogMatcher) Match(
	_ context.Context,
	text string,
	intents []apppresenter.IntentDefinition,
) (MatchResult, error) {
	normalizedQuery := normalizeText(text)
	queryTokens := tokenSet(normalizedQuery)
	if len(queryTokens) == 0 {
		return MatchResult{}, nil
	}

	candidates := make([]Candidate, 0, len(intents))
	for _, intentDefinition := range intents {
		bestScore := 0.0
		for _, example := range intentDefinition.Examples {
			score := scoreExample(normalizedQuery, queryTokens, normalizeText(example))
			if score > bestScore {
				bestScore = score
			}
		}
		if bestScore == 0 {
			continue
		}
		candidates = append(candidates, Candidate{
			IntentKey:  intentDefinition.Key,
			Confidence: bestScore,
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Confidence == candidates[j].Confidence {
			return candidates[i].IntentKey < candidates[j].IntentKey
		}
		return candidates[i].Confidence > candidates[j].Confidence
	})

	if len(candidates) == 0 {
		return MatchResult{}, nil
	}

	limit := 3
	if len(candidates) < limit {
		limit = len(candidates)
	}

	return MatchResult{
		IntentKey:  candidates[0].IntentKey,
		Confidence: candidates[0].Confidence,
		Candidates: append([]Candidate(nil), candidates[:limit]...),
	}, nil
}

func normalizeText(text string) string {
	lowered := strings.ToLower(strings.ReplaceAll(text, "ё", "е"))
	var builder strings.Builder
	builder.Grow(len(lowered))

	lastSpace := false
	for _, r := range lowered {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			builder.WriteByte(' ')
			lastSpace = true
		}
	}

	return strings.TrimSpace(builder.String())
}

func NormalizeForSeed(text string) string {
	return normalizeText(text)
}

func tokenSet(text string) map[string]struct{} {
	if text == "" {
		return nil
	}

	tokens := strings.Fields(text)
	set := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		if len(token) < 2 {
			continue
		}
		set[token] = struct{}{}
	}
	return set
}

func scoreExample(query string, queryTokens map[string]struct{}, example string) float64 {
	if example == "" || len(queryTokens) == 0 {
		return 0
	}
	if query == example {
		return 1
	}

	exampleTokens := tokenSet(example)
	if len(exampleTokens) == 0 {
		return 0
	}

	overlap := 0
	for token := range queryTokens {
		if _, ok := exampleTokens[token]; ok {
			overlap++
		}
	}
	if overlap == 0 {
		return 0
	}

	maxTokenCount := len(queryTokens)
	if len(exampleTokens) > maxTokenCount {
		maxTokenCount = len(exampleTokens)
	}

	score := float64(overlap) / float64(maxTokenCount)
	if strings.Contains(query, example) || strings.Contains(example, query) {
		score += 0.2
	}
	if score > 1 {
		score = 1
	}
	return score
}
