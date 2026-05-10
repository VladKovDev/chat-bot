package normalization

import (
	"context"

	"github.com/VladKovDev/chat-bot/pkg/logger"
)

type LemmatizerPort interface {
	Lemmatize(ctx context.Context, tokens []string) ([]string, error)
}

type lemmatizerStep struct {
	port   LemmatizerPort
	logger logger.Logger
}

func newLemmatizerStep(port LemmatizerPort, logger logger.Logger) *lemmatizerStep {
	return &lemmatizerStep{port: port, logger: logger}
}

func (s *lemmatizerStep) lemmatize(ctx context.Context, tokens []string) []string {
	if len(tokens) == 0 {
		return tokens
	}

	lemmas, err := s.port.Lemmatize(ctx, tokens)
	if err != nil {
		s.logger.Warn("lemmatizer unavailable, using original tokens",
			s.logger.String("error", err.Error()),
			s.logger.Int("token_count", len(tokens)),
		)
		return tokens
	}

	return lemmas
}
