package rule_based

import (
	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

type Intent struct {
	Name  string `yaml:"name"`
	Rules []Rule `yaml:"rules"`
}

type Threshold struct {
	MinConfidence  float64 `yaml:"min_confidence"`
	AmbiguityDelta float64 `yaml:"ambiguity_delta"`
}

type Config struct {
	Intents   []Intent  `yaml:"intents"`
	Threshold Threshold `yaml:"threshold"`
}

type RuleBased struct {
	cfg    Config
	logger logger.Logger
}

func NewRuleBased(cfg Config, logger logger.Logger) *RuleBased {
	return &RuleBased{
		cfg: cfg,
		logger: logger,
	}
}

func (r *RuleBased) Classify(text string) (conversation.Event, error) {
	scores := make(map[string]float64)
	matched := make(map[string][]string)
	maxPossible := make(map[string]float64)

	for _, intent := range r.cfg.Intents {
		for _, rule := range intent.Rules {
			for range rule.Values {
				maxPossible[intent.Name] += rule.Weight
			}
			hits, score := rule.Match(text)
			scores[intent.Name] += score
			matched[intent.Name] = append(matched[intent.Name], hits...)
		}
	}
	r.logger.Debug("classification scores", r.logger.Any("scores", scores), r.logger.Any("matched", matched))

	best, second := topTwo(scores)

	confidence := best.score / maxPossible[best.name]
	r.logger.Debug("classification confidence", r.logger.Any("confidence", confidence))
	if best.score == 0 || confidence < r.cfg.Threshold.MinConfidence {
		return conversation.EventUnknown, nil
	}

	if best.score-second.score < r.cfg.Threshold.AmbiguityDelta*maxPossible[best.name] {
		// TODO logging ambiguity
	}

	return conversation.Event(best.name), nil
}

func topTwo(scores map[string]float64) (best, second struct {
	name  string
	score float64
}) {
	for name, score := range scores {
		if score > best.score {
			second = best
			best = struct {
				name  string
				score float64
			}{name, score}
		} else if score > second.score {
			second = struct {
				name  string
				score float64
			}{name, score}
		}
	}
	return best, second
}
