package nlp

import (
	"github.com/VladKovDev/chat-bot/pkg/logger"
	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/nlp/normalization"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/nlp/rule_based"
)

type Classifier struct {
	RuleBased *rule_based.RuleBased
	logger    logger.Logger
}

func NewClassifier(ruleBasedClassifier *rule_based.RuleBased, logger logger.Logger) *Classifier {
	return &Classifier{
		RuleBased: ruleBasedClassifier,
		logger: logger,
	}
}

func (c *Classifier) Classify(textRow string) (conversation.Event, error) {
	text := normalization.Normalize(textRow)

	event, err := c.RuleBased.Classify(text)
	if err != nil {
		return conversation.Event(""), err
	}
	if event == conversation.EventUnknown {
		// TODO embeddings-based classifier
	}

	c.logger.Debug("event classified", c.logger.Any("event", event))
	return event, nil
}
