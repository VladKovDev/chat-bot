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

var normalizedTokenAliases = map[string]string{
	"account":     "аккаунт",
	"appointment": "запись",
	"booking":     "запись",
	"but":         "",
	"call":        "",
	"cancel":      "отмена",
	"code":        "код",
	"complaint":   "жалоба",
	"contact":     "контакты",
	"contacts":    "контакты",
	"down":        "не работает",
	"failed":      "не прошла",
	"faq":         "faq",
	"forgot":      "забыл",
	"general":     "общий",
	"help":        "помощь",
	"hours":       "часы",
	"inactive":    "не активировалась",
	"info":        "информация",
	"is":          "",
	"list":        "список",
	"location":    "адрес",
	"login":       "логин",
	"master":      "мастер",
	"missing":     "не приходит",
	"my":          "",
	"operator":    "оператор",
	"paid":        "оплата прошла",
	"password":    "пароль",
	"payment":     "платеж",
	"please":      "",
	"premises":    "помещение",
	"price":       "цена",
	"prices":      "цены",
	"problem":     "проблема",
	"question":    "вопрос",
	"random":      "",
	"refund":      "возврат",
	"rent":        "аренда",
	"rules":       "правила",
	"service":     "услуга",
	"services":    "услуги",
	"site":        "сайт",
	"sms":         "смс",
	"status":      "статус",
	"text":        "текст",
	"unclear":     "непонятно",
	"workspace":   "коворкинг",
}

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
	return rankCandidates(candidates), nil
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

	normalized := strings.TrimSpace(builder.String())
	if normalized == "" {
		return ""
	}
	return canonicalizeSupportTokens(normalized)
}

func NormalizeForSeed(text string) string {
	return normalizeText(text)
}

func canonicalizeSupportTokens(text string) string {
	tokens := strings.Fields(text)
	if len(tokens) == 0 {
		return ""
	}

	normalized := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if alias, ok := normalizedTokenAliases[token]; ok {
			if alias == "" {
				continue
			}
			normalized = append(normalized, strings.Fields(alias)...)
			continue
		}
		normalized = append(normalized, token)
	}
	return strings.Join(normalized, " ")
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

type lexicalScoreBreakdown struct {
	Total           float64
	TokenOverlap    float64
	FuzzyToken      float64
	Trigram         float64
	SubstringBonus  float64
	IdentifierBonus float64
}

func scoreExample(query string, queryTokens map[string]struct{}, queryRaw string, example string, exampleRaw string) lexicalScoreBreakdown {
	if example == "" || query == "" {
		return lexicalScoreBreakdown{}
	}
	if query == example {
		return lexicalScoreBreakdown{Total: 1}
	}

	exampleTokens := tokenSet(example)
	breakdown := lexicalScoreBreakdown{
		TokenOverlap: tokenOverlapScore(queryTokens, exampleTokens),
		FuzzyToken:   fuzzyTokenScore(strings.Fields(query), strings.Fields(example)),
		Trigram:      trigramJaccard(query, example),
	}

	score := breakdown.TokenOverlap*0.5 + breakdown.FuzzyToken*0.3 + breakdown.Trigram*0.2
	if strings.Contains(query, example) || strings.Contains(example, query) {
		breakdown.SubstringBonus = 0.2
		score += breakdown.SubstringBonus
	}
	if identifierType := lexicalIdentifierType(queryRaw); identifierType != "" && identifierType == lexicalIdentifierType(exampleRaw) {
		breakdown.IdentifierBonus = 0.18
		score += breakdown.IdentifierBonus
	}
	if score > 1 {
		score = 1
	}
	breakdown.Total = score
	return breakdown
}

func lexicalIntentCandidates(text string, intents []apppresenter.IntentDefinition, limit int) []Candidate {
	normalizedQuery := normalizeText(text)
	queryTokens := tokenSet(normalizedQuery)
	if normalizedQuery == "" {
		return nil
	}

	candidates := make([]Candidate, 0, len(intents))
	for _, intentDefinition := range intents {
		best := lexicalScoreBreakdown{}
		bestExample := ""
		for _, example := range intentDefinition.Examples {
			normalizedExample := normalizeText(example)
			score := scoreExample(normalizedQuery, queryTokens, text, normalizedExample, example)
			if score.Total > best.Total {
				best = score
				bestExample = example
			}
		}
		if best.Total == 0 {
			continue
		}
		candidates = append(candidates, Candidate{
			IntentKey:  intentDefinition.Key,
			Confidence: best.Total,
			Source:     CandidateSourceLexicalFuzzy,
			Text:       bestExample,
			Metadata: map[string]any{
				"category": intentDefinition.Category,
				"score_components": map[string]float64{
					"token_overlap":    best.TokenOverlap,
					"fuzzy_token":      best.FuzzyToken,
					"trigram":          best.Trigram,
					"substring_bonus":  best.SubstringBonus,
					"identifier_bonus": best.IdentifierBonus,
				},
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

func lexicalIdentifierType(text string) string {
	switch {
	case bookingIdentifierPattern.MatchString(text):
		return "booking_number"
	case workspaceIdentifierPattern.MatchString(text):
		return "workspace_booking"
	case paymentIdentifierPattern.MatchString(text):
		return "payment_id"
	case userIdentifierPattern.MatchString(text):
		return "user_id"
	case emailIdentifierPattern.MatchString(text):
		return "email"
	case phoneIdentifierPattern.MatchString(text):
		return "phone"
	default:
		return ""
	}
}
