package decision

import (
	"context"
	"math"
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
	candidates := lexicalIntentCandidates(text, intents, 3)
	if len(candidates) == 0 {
		return MatchResult{}, nil
	}

	return MatchResult{
		IntentKey:  candidates[0].IntentKey,
		Confidence: candidates[0].Confidence,
		Candidates: append([]Candidate(nil), candidates...),
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
	if example == "" || query == "" {
		return 0
	}
	if query == example {
		return 1
	}

	exampleTokens := tokenSet(example)
	tokenScore := tokenOverlapScore(queryTokens, exampleTokens)
	fuzzyTokenScore := fuzzyTokenScore(strings.Fields(query), strings.Fields(example))
	trigramScore := trigramJaccard(query, example)

	score := tokenScore*0.5 + fuzzyTokenScore*0.3 + trigramScore*0.2
	if strings.Contains(query, example) || strings.Contains(example, query) {
		score += 0.2
	}
	if score > 1 {
		score = 1
	}
	return score
}

func lexicalIntentCandidates(text string, intents []apppresenter.IntentDefinition, limit int) []Candidate {
	normalizedQuery := normalizeText(text)
	queryTokens := tokenSet(normalizedQuery)
	if normalizedQuery == "" {
		return nil
	}

	candidates := make([]Candidate, 0, len(intents))
	for _, intentDefinition := range intents {
		bestScore := 0.0
		bestExample := ""
		for _, example := range intentDefinition.Examples {
			normalizedExample := normalizeText(example)
			score := scoreExample(normalizedQuery, queryTokens, normalizedExample)
			if score > bestScore {
				bestScore = score
				bestExample = example
			}
		}
		if bestScore == 0 {
			continue
		}
		candidates = append(candidates, Candidate{
			IntentKey:  intentDefinition.Key,
			Confidence: bestScore,
			Source:     CandidateSourceLexicalFuzzy,
			Text:       bestExample,
			Metadata: map[string]any{
				"category": intentDefinition.Category,
			},
		})
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

func tokenOverlapScore(queryTokens, exampleTokens map[string]struct{}) float64 {
	if len(queryTokens) == 0 || len(exampleTokens) == 0 {
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

	return float64(overlap) / float64(maxTokenCount)
}

func fuzzyTokenScore(queryTokens, exampleTokens []string) float64 {
	if len(queryTokens) == 0 || len(exampleTokens) == 0 {
		return 0
	}

	total := 0.0
	for _, queryToken := range queryTokens {
		best := 0.0
		for _, exampleToken := range exampleTokens {
			score := tokenSimilarity(queryToken, exampleToken)
			if score > best {
				best = score
			}
		}
		total += best
	}

	return total / float64(len(queryTokens))
}

func tokenSimilarity(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}
	if strings.HasPrefix(a, b) || strings.HasPrefix(b, a) {
		shorter := len(a)
		if len(b) < shorter {
			shorter = len(b)
		}
		longer := len(a)
		if len(b) > longer {
			longer = len(b)
		}
		return float64(shorter) / float64(longer)
	}
	return trigramJaccard(a, b)
}

func trigramJaccard(a, b string) float64 {
	aSet := trigramSet(a)
	bSet := trigramSet(b)
	if len(aSet) == 0 || len(bSet) == 0 {
		return 0
	}

	intersection := 0
	for token := range aSet {
		if _, ok := bSet[token]; ok {
			intersection++
		}
	}
	union := len(aSet) + len(bSet) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func trigramSet(text string) map[string]struct{} {
	runes := []rune(text)
	if len(runes) < 3 {
		if len(runes) == 0 {
			return nil
		}
		return map[string]struct{}{text: {}}
	}

	set := make(map[string]struct{}, len(runes)-2)
	for i := 0; i <= len(runes)-3; i++ {
		set[string(runes[i:i+3])] = struct{}{}
	}
	return set
}

func normalizeConfidence(value float64) float64 {
	return math.Max(0, math.Min(1, value))
}
