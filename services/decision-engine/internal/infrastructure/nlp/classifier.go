package nlp

import (
	"context"

	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/nlp/normalization"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/nlp/rule_based"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

type Classifier struct {
	RuleBased  *rule_based.RuleBased
	Normalizer *normalization.Pipeline
	logger     logger.Logger
}

func NewClassifier(ruleBasedClassifier *rule_based.RuleBased, normalizer *normalization.Pipeline, logger logger.Logger) *Classifier {
	return &Classifier{
		RuleBased:  ruleBasedClassifier,
		Normalizer: normalizer,
		logger:     logger,
	}
}

func (c *Classifier) Classify(ctx context.Context, textRow string) (state.Event, error) {
	textTokens := c.Normalizer.Normalize(ctx, textRow)

	c.logger.Debug("text normalized", c.logger.Any("tokens", textTokens))

	event, err := c.RuleBased.Classify(textTokens)
	if err != nil {
		return state.Event(""), err
	}
	if event == state.EventUnknown {
		// TODO embeddings-based classifier
	}

	c.logger.Debug("event classified", c.logger.Any("event", event))
	return event, nil
}
