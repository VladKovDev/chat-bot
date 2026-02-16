package app

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/VladKovDev/chat-bot/internal/config"
	loggerCfg "github.com/VladKovDev/chat-bot/internal/config/logger"
	postgresCfg "github.com/VladKovDev/chat-bot/internal/config/postgres"
	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository/postgres"
	"github.com/VladKovDev/chat-bot/internal/transport/telegram_temp"
	"github.com/VladKovDev/chat-bot/internal/worker"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

type App struct {
	LoggerConfig   *logger.Config
	PostgresConfig *postgres.Config

	Logger logger.Logger
	DB     *postgres.Pool

	ConversationRepo    conversation.Repository
	ConversationService *conversation.Service

	Bot *telegram_temp.Bot
}

func NewApp(loggerConfig *logger.Config,
	postgresConfig *postgres.Config,
	logger logger.Logger,
	db *postgres.Pool) *App {

	var conversationRepo = postgres.NewConversationRepo(db)
	var conversationService = conversation.NewService(conversationRepo)

	return &App{
		LoggerConfig:   loggerConfig,
		PostgresConfig: postgresConfig,
		Logger:         logger,
		DB:             db,

		ConversationRepo:    conversationRepo,
		ConversationService: conversationService,
	}
}

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

	postgresConfig, err := postgresCfg.LoadConfig(viper)
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
	pool, err := postgres.NewPool(ctx, &postgresConfig, logger)
	if err != nil {
		return fmt.Errorf("failed to init database: %w", err)
	}

	app := NewApp(&loggerConfig, &postgresConfig, logger, pool)

	// Initialize message worker
	msgWorker := worker.NewMessageWorker(app.ConversationService)

	// Initialize Telegram bot
	bot, err := telegram_temp.NewBot(os.Getenv("TELEGRAM_BOT_TOKEN"), msgWorker, app.ConversationService)
	if err != nil {
		return fmt.Errorf("failed to create telegram bot: %w", err)
	}
	app.Bot = bot

	// Start bot in goroutine
	go func() {
		if err := bot.Start(ctx); err != nil {
			logger.Error("bot stopped with error", logger.Err(err))
		}
	}()

	logger.Info("application started")

	// Graceful shutdown
	if err := GracefulShutdown(ctx, 30*time.Second, logger, app.Bot, app.DB); err != nil {
		logger.Error("error during shutdown", logger.Err(err))
		return err
	}

	logger.Info("application stopped successfully")
	return nil
}
