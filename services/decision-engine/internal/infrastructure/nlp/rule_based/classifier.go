package rule_based

import (
	"fmt"
	"strings"

	"github.com/VladKovDev/chat-bot/internal/domain/intent"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

type RuleType string

const (
	RuleKeyword RuleType = "keyword"
	RulePhrase  RuleType = "phrase"
)

type Rule struct {
	Type   RuleType
	Weight float64
	Values []string
}

type scoredIntent struct {
	name  string
	score float64
}

type compiledValue struct {
	id     int
	intent string
	raw    string
	weight float64
	tokens []string // only for phrase
}

type RuleBased struct {
	cfg    Config
	logger logger.Logger

	values       []compiledValue
	keywordIndex map[string][]int // token -> compiledValue IDs
	phraseIndex  map[string][]int // first token -> compiledValue IDs
	maxPossible  map[string]float64
}

func NewRuleBased(cfg Config, logger logger.Logger) (*RuleBased, error) {
	r := &RuleBased{
		cfg:    cfg,
		logger: logger,
	}
	if err := r.compile(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *RuleBased) compile() error {
	if r.values != nil {
		return nil
	}

	r.maxPossible = make(map[string]float64, len(r.cfg.Intents))
	r.keywordIndex = make(map[string][]int)
	r.phraseIndex = make(map[string][]int)

	// rough capacity to reduce reallocations
	capHint := 0
	for _, intent := range r.cfg.Intents {
		for _, rule := range intent.Rules {
			capHint += len(rule.Values)
		}
	}
	r.values = make([]compiledValue, 0, capHint)

	id := 0
	for _, intent := range r.cfg.Intents {
		for _, rule := range intent.Rules {
			for _, raw := range rule.Values {
				if raw == "" {
					continue
				}

				item := compiledValue{
					id:     id,
					intent: intent.Name,
					raw:    raw,
					weight: rule.Weight,
				}

				switch rule.Type {
				case RuleKeyword:
					r.keywordIndex[raw] = append(r.keywordIndex[raw], id)

				case RulePhrase:
					item.tokens = strings.Fields(raw)
					if len(item.tokens) == 0 {
						continue
					}
					r.phraseIndex[item.tokens[0]] = append(r.phraseIndex[item.tokens[0]], id)

				default:
					return fmt.Errorf("unsupported rule type %q", rule.Type)
				}

				r.values = append(r.values, item)
				r.maxPossible[intent.Name] += rule.Weight
				id++
			}
		}
	}

	return nil
}

func (r *RuleBased) Classify(tokens []string) (intent.Intent, error) {
	if err := r.compile(); err != nil {
		r.logger.Error("failed to compile rule-based classifier", r.logger.Err(err))
		return intent.IntentUnknown, err
	}
	if len(tokens) == 0 {
		return intent.IntentUnknown, nil
	}

	tokenSet := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		tokenSet[t] = struct{}{}
	}

	scores := make(map[string]float64, len(r.cfg.Intents))
	matched := make(map[string][]string, len(r.cfg.Intents))

	seen := make(map[int]struct{}, 16)

	for token := range tokenSet {
		ids := r.keywordIndex[token]
		for _, id := range ids {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}

			item := r.values[id]
			scores[item.intent] += item.weight
			matched[item.intent] = append(matched[item.intent], item.raw)
		}
	}

	// Phrase matching: first-token bucket + exact subsequence comparison.
	for i, token := range tokens {
		ids := r.phraseIndex[token]
		if len(ids) == 0 {
			continue
		}

		for _, id := range ids {
			if _, ok := seen[id]; ok {
				continue
			}

			item := r.values[id]
			n := len(item.tokens)
			if n == 0 || i+n > len(tokens) {
				continue
			}

			if equalTokens(tokens[i:i+n], item.tokens) {
				seen[id] = struct{}{}
				scores[item.intent] += item.weight
				matched[item.intent] = append(matched[item.intent], item.raw)
			}
		}
	}

	r.logger.Debug(
		"classification scores",
		r.logger.Any("scores", scores),
		r.logger.Any("matched", matched),
	)

	best, second := topTwo(scores)
	if best.name == "" || best.score == 0 {
		return intent.IntentUnknown, nil
	}

	max := r.maxPossible[best.name]
	if max <= 0 {
		return intent.IntentUnknown, nil
	}

	confidence := best.score / max
	r.logger.Debug("classification confidence", r.logger.Any("confidence", confidence))

	if confidence < r.cfg.Threshold.MinConfidence {
		return intent.IntentUnknown, nil
	}

	if second.name != "" && best.score-second.score < r.cfg.Threshold.AmbiguityDelta*max {
		r.logger.Debug(
			"classification ambiguity",
			r.logger.Any("best", best.name),
			r.logger.Any("best_score", best.score),
			r.logger.Any("second", second.name),
			r.logger.Any("second_score", second.score),
		)
	}

	return intent.Intent(best.name), nil
}

func equalTokens(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range b {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func topTwo(scores map[string]float64) (best, second scoredIntent) {
	for name, score := range scores {
		if score > best.score {
			second = best
			best = scoredIntent{name: name, score: score}
			continue
		}
		if score > second.score {
			second = scoredIntent{name: name, score: score}
		}
	}
	return best, second
}
