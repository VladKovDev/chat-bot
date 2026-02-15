package database

import (
	"time"

	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository"
)

func SetDefaultConfig() repository.Config {
	return repository.Config{
		Host:              "localhost",
		Port:              5432,
		User:              "postgres",
		Password:          "",
		Name:              "chat-bot",
		SSLMode:           "require",
		MaxOpenConns:      10,
		MaxIdleConns:      5,
		ConnMaxLifetime:   1 * time.Hour,
		ConnMaxIdleTime:   15 * time.Minute,
		HealthCheckPeriod: 1 * time.Minute,
	}
}
