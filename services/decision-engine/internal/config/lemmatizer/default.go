package lemmatizer

import (
	"time"

	"github.com/VladKovDev/chat-bot/internal/infrastructure/lemmatizer"
)

func SetDefaultConfig() lemmatizer.Config {
	return lemmatizer.Config{
		BaseURL:       "http://localhost:8000",
		Timeout:       5 * time.Second,
		CBMaxRequests: 3,
		CBInterval:    1 * time.Minute,
		CBMaxFailures: 5,
		CBTimeout:     30 * time.Second,
	}
}

