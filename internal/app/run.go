package app

import (
	"context"
	"fmt"
	"os"

	"github.com/VladKovDev/chat-bot/internal/config"
	loggerCfg "github.com/VladKovDev/chat-bot/internal/config/logger"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

func Run(ctx context.Context) error {
	configPath := os.Getenv("CONFIG_PATH")
	appEnv := os.Getenv("APP_ENV")

	// Initialize configuration
	viper, err := config.Init(configPath, appEnv)
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	loggerConfig, err := loggerCfg.LoadConfig(viper)
	if err != nil {
		return fmt.Errorf("failed to load logger config: %w", err)
	}

	// Initialize logger
	logger, err := logger.New(loggerConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	logger.Debug("logger debug enabled")

	_ = loggerConfig
	return nil
}
