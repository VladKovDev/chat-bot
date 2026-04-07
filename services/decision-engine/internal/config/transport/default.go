package transport

import (
	"time"

	"github.com/VladKovDev/chat-bot/internal/transport/http"
)

func SetDefaultConfig() http.Config {
	return http.Config{
		Address:         ":8080",
		ReadTimeout:     10 * time.Second,
		ReadHeadTimeout: 5 * time.Second,
		WriteTimeout:    10 * time.Second,
		IdleTimeout:     60 * time.Second,
		MaxHeaderBytes:  1 << 20, // 1 MB

		Timeout:        15 * time.Second,
		BodyLimit:      10 * 1024 * 1024, // 10 MB
		EnableLogs:     true,
		EnableRecovery: true,
	}
}