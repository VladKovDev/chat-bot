package llm

import (
	"time"

	"github.com/VladKovDev/chat-bot/internal/infrastructure/llm"
)

func SetDefaultConfig() llm.Config {
	return llm.Config{
		BaseURL:       "http://localhost:8001",
		Timeout:       10 * time.Second,
		CBMaxRequests: 3,
		CBInterval:    1 * time.Minute,
		CBMaxFailures: 5,
		CBTimeout:     30 * time.Second,
	}
}