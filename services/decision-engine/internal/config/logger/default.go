package logger

import "github.com/VladKovDev/chat-bot/pkg/logger"

func SetDefaultConfig() logger.Config {
	return logger.Config{
		Level:        "info",
		Format:       "json",
		Output:       "stdout",
		EnableColors: false,
		FilePath:     "",
		MaxSize:      0,
		MaxBackups:   0,
		MaxAge:       0,
		Compress:     false,
	}
}
