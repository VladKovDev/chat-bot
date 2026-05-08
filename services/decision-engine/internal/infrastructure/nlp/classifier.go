package nlp

import (
	"context"

	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/intent"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/nlp/normalization"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/nlp/rule_based"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

type Classifier struct {
	RuleBased   *rule_based.RuleBased
	Normalizer  *normalization.Pipeline
	EventAdapter *EventAdapter
	logger      logger.Logger
}

func NewClassifier(
	ruleBasedClassifier *rule_based.RuleBased,
	normalizer *normalization.Pipeline,
	eventAdapter *EventAdapter,
	logger logger.Logger,
) *Classifier {
	return &Classifier{
		RuleBased:    ruleBasedClassifier,
		Normalizer:   normalizer,
		EventAdapter: eventAdapter,
		logger:       logger,
	}
}

func (c *Classifier) Classify(ctx context.Context, textRow string) (session.Event, error) {
	textTokens := c.Normalizer.Normalize(ctx, textRow)

	c.logger.Debug("text normalized", c.logger.Any("tokens", textTokens))

	userIntent, err := c.RuleBased.Classify(textTokens)
	if err != nil {
		return session.Event(""), err
	}
	if userIntent == intent.IntentUnknown {
		// TODO embeddings-based classifier
	}

	// Map intent to event
	event := c.EventAdapter.IntentToEvent(userIntent)

	c.logger.Debug("classified",
		c.logger.Any("intent", userIntent),
		c.logger.Any("event", event))

	return event, nil
}
