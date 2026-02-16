package app

import (
	"context"
	"fmt"
	"os"

	"github.com/VladKovDev/chat-bot/internal/config"
	databaseCfg "github.com/VladKovDev/chat-bot/internal/config/database"
	loggerCfg "github.com/VladKovDev/chat-bot/internal/config/logger"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository/postgres"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/telegram"
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

	databaseConfig, err := databaseCfg.LoadConfig(viper)
	if err != nil {
		return fmt.Errorf("failed to load database config: %w", err)
	}

	// Initialize logger
	logger, err := logger.New(loggerConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	logger.Debug("logger debug enabled")

	// Initialize DB
	pool, err := postgres.NewPool(ctx, &databaseConfig, logger)
	if err != nil {
		return fmt.Errorf("failed to init database: %w", err)
	}

	bot, err := telegram.NewBot(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if err != nil {
		return fmt.Errorf("failed to create telegram bot: %w", err)
	}

	if err := bot.Start(); err != nil {
		logger.Error("failed to start telegram bot", logger.Err(err))
	}

	_ = pool
	_ = loggerConfig
	return nil
}
