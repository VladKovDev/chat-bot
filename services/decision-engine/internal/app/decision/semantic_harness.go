package decision

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"sort"
	"strings"

	apppresenter "github.com/VladKovDev/chat-bot/internal/app/presenter"
)

const corpusHarnessEmbeddingDimension = 384

type deterministicHashEmbedder struct {
	dimension int
	seed      string
}

func newDeterministicHashEmbedder(dimension int, seed string) *deterministicHashEmbedder {
	return &deterministicHashEmbedder{
		dimension: dimension,
		seed:      seed,
	}
}

func (e *deterministicHashEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	values := make([]float64, e.dimension)
	for _, item := range embeddingFeatures(text) {
		digest := sha256.Sum256([]byte(e.seed + ":" + item.feature))
		index := int(digest[0])<<24 | int(digest[1])<<16 | int(digest[2])<<8 | int(digest[3])
		index = index % e.dimension
		sign := 1.0
		if digest[4]%2 != 0 {
			sign = -1.0
		}
		values[index] += sign * item.weight
	}
	return normalizeVector(values), nil
}

type inMemoryIntentSearchRepository struct {
	rows []inMemoryIntentVector
}

type inMemoryIntentVector struct {
	embedding []float64
	result    IntentSearchResult
}

func newInMemoryIntentSearchRepository(
	ctx context.Context,
	embedder Embedder,
	intents []apppresenter.IntentDefinition,
) (*inMemoryIntentSearchRepository, error) {
	rows := make([]inMemoryIntentVector, 0)
	for _, intentDef := range intents {
		for _, example := range intentDef.Examples {
			normalized := normalizeText(example)
			if normalized == "" {
				continue
			}
			embedding, err := embedder.Embed(ctx, example)
			if err != nil {
				return nil, fmt.Errorf("embed in-memory example %s: %w", intentDef.Key, err)
			}
			rows = append(rows, inMemoryIntentVector{
				embedding: embedding,
				result: IntentSearchResult{
					IntentKey:      intentDef.Key,
					Category:       intentDef.Category,
					ResponseKey:    intentDef.ResponseKey,
					Text:           example,
					NormalizedText: normalized,
					Locale:         "ru",
					Weight:         1,
				},
			})
		}
	}
	return &inMemoryIntentSearchRepository{rows: rows}, nil
}

func (r *inMemoryIntentSearchRepository) SearchIntentExamples(
	_ context.Context,
	embedding []float64,
	locale string,
	limit int,
) ([]IntentSearchResult, error) {
	if limit <= 0 {
		limit = 3
	}

	rows := make([]IntentSearchResult, 0, len(r.rows))
	for _, row := range r.rows {
		if locale != "" && row.result.Locale != locale {
			continue
		}
		result := row.result
		result.Confidence = cosineSimilarity(embedding, row.embedding)
		rows = append(rows, result)
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Confidence == rows[j].Confidence {
			if rows[i].IntentKey == rows[j].IntentKey {
				return rows[i].NormalizedText < rows[j].NormalizedText
			}
			return rows[i].IntentKey < rows[j].IntentKey
		}
		return rows[i].Confidence > rows[j].Confidence
	})

	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

func embeddingFeatures(text string) []struct {
	feature string
	weight  float64
} {
	tokens := strings.Fields(normalizeText(text))
	features := make([]struct {
		feature string
		weight  float64
	}, 0, len(tokens)*8)
	for _, token := range tokens {
		features = append(features, struct {
			feature string
			weight  float64
		}{feature: "tok:" + token, weight: 2})
		for _, ngram := range charNgrams(token) {
			features = append(features, struct {
				feature string
				weight  float64
			}{feature: "ng:" + ngram, weight: 1})
		}
	}
	for index := 0; index < len(tokens)-1; index++ {
		features = append(features, struct {
			feature string
			weight  float64
		}{feature: "bi:" + tokens[index] + "_" + tokens[index+1], weight: 1.5})
	}
	return features
}

func charNgrams(token string) []string {
	runes := []rune(token)
	if len(runes) <= 3 {
		return []string{token}
	}

	ngrams := make([]string, 0, len(runes)*3)
	for _, size := range []int{3, 4, 5} {
		if len(runes) < size {
			continue
		}
		for index := 0; index <= len(runes)-size; index++ {
			ngrams = append(ngrams, string(runes[index:index+size]))
		}
	}
	return ngrams
}

func normalizeVector(values []float64) []float64 {
	norm := math.Sqrt(sumSquares(values))
	if norm == 0 {
		return values
	}
	normalized := make([]float64, len(values))
	for index, value := range values {
		normalized[index] = value / norm
	}
	return normalized
}

func sumSquares(values []float64) float64 {
	total := 0.0
	for _, value := range values {
		total += value * value
	}
	return total
}

func cosineSimilarity(left, right []float64) float64 {
	if len(left) == 0 || len(left) != len(right) {
		return 0
	}
	total := 0.0
	for index := range left {
		total += left[index] * right[index]
	}
	if total < 0 {
		return 0
	}
	if total > 1 {
		return 1
	}
	return total
}
