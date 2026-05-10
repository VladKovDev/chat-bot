package nlp

import (
	"time"

	infranlp "github.com/VladKovDev/chat-bot/internal/infrastructure/nlp"
)

func SetDefaultConfig() infranlp.EmbedderConfig {
	return infranlp.EmbedderConfig{
		BaseURL:           "http://localhost:8081",
		Timeout:           5 * time.Second,
		ExpectedDimension: 384,
	}
}
