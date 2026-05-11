package nlp

import (
	"time"

	appseed "github.com/VladKovDev/chat-bot/internal/app/seed"
	infranlp "github.com/VladKovDev/chat-bot/internal/infrastructure/nlp"
)

func SetDefaultConfig() infranlp.EmbedderConfig {
	return infranlp.EmbedderConfig{
		BaseURL:           "http://localhost:8081",
		Timeout:           5 * time.Second,
		ExpectedDimension: appseed.SemanticEmbeddingDimension,
	}
}
