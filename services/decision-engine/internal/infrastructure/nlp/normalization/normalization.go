package normalization

import (
	"context"
	"time"

	"github.com/VladKovDev/chat-bot/pkg/logger"
)

type Pipeline struct {
	lemma             *lemmatizerStep
	lemmatizerTimeout time.Duration
	logger            logger.Logger
}

func NewPipeline(port LemmatizerPort, lemmatizerTimeout time.Duration, logger logger.Logger) *Pipeline {
	return &Pipeline{
		lemma:             newLemmatizerStep(port, logger),
		logger:            logger,
		lemmatizerTimeout: lemmatizerTimeout,
	}
}

func (p *Pipeline) Normalize(ctx context.Context, text string) []string {
	// Stage 1: Tokenize
	// NFC normalization + lowercase + regex word extraction.
	tokens := Tokenize(text)
	if len(tokens) == 0 {
		return []string{}
	}

	// Stage 2: Remove stopwords
	tokens = RemoveStopwords(tokens)
	if len(tokens) == 0 {
		return []string{}
	}

	// Stage 3: Lemmatize
	lctx, cancel := context.WithTimeout(ctx, p.lemmatizerTimeout)
	defer cancel()

	tokens = p.lemma.lemmatize(lctx, tokens)

	// Stage 4: MWE
	tokens = ApplyMWE(tokens)

	return tokens
}

// Normalize is a package-level convenience function for callers that don't
// need pipeline configuration (tests, scripts, simple CLIs).

func Normalize(text string) []string {
	tokens := Tokenize(text)
	if len(tokens) == 0 {
		return []string{}
	}
	tokens = RemoveStopwords(tokens)
	tokens = ApplyMWE(tokens)
	return tokens
}
