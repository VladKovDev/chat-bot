package app

import (
	"context"
	"fmt"
	"os"
	"time"

	appactions "github.com/VladKovDev/chat-bot/internal/app/actions"
	apppresenter "github.com/VladKovDev/chat-bot/internal/app/presenter"
	appprocessor "github.com/VladKovDev/chat-bot/internal/app/processor"
	appworker "github.com/VladKovDev/chat-bot/internal/app/worker"
	"github.com/VladKovDev/chat-bot/internal/config"
	llmCfg "github.com/VladKovDev/chat-bot/internal/config/llm"
	loggerCfg "github.com/VladKovDev/chat-bot/internal/config/logger"
	postgresCfg "github.com/VladKovDev/chat-bot/internal/config/postgres"
	transportCfg "github.com/VladKovDev/chat-bot/internal/config/transport"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/message"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/user"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/llm"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository/postgres"
	"github.com/VladKovDev/chat-bot/internal/transport/http"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

type App struct {
	LoggerConfig   *logger.Config
	PostgresConfig *postgres.Config

	Logger    logger.Logger
	DB        *postgres.Pool
	LLMClient *llm.Client
	HTTP      *http.Server

	ConversationRepo session.Repository
	MessageRepo      message.Repository
	UserRepo         user.Repository
	ActionLogRepo    action.LogRepository
	Worker           *appworker.MessageWorker
}

func NewApp(
	loggerConfig *logger.Config,
	postgresConfig *postgres.Config,
	logger logger.Logger,
	db *postgres.Pool,
	llmClient *llm.Client,
	httpServer *http.Server,
	worker *appworker.MessageWorker,
	sessionRepo session.Repository,
	messageRepo message.Repository,
	userRepo user.Repository,
	actionLogRepo action.LogRepository,
) *App {

	return &App{
		LoggerConfig:     loggerConfig,
		PostgresConfig:   postgresConfig,
		Logger:           logger,
		DB:               db,
		LLMClient:        llmClient,
		HTTP:             httpServer,
		Worker:           worker,
		ConversationRepo: sessionRepo,
		MessageRepo:      messageRepo,
		UserRepo:         userRepo,
		ActionLogRepo:    actionLogRepo,
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

	llmConfig, err := llmCfg.LoadConfig(viper)
	if err != nil {
		return fmt.Errorf("failed to load llm config: %w", err)
	}

	httpConfig, err := transportCfg.LoadConfig(viper)
	if err != nil {
		return fmt.Errorf("failed to load http config: %w", err)
	}

	// Initialize logger
	logger, err := logger.New(loggerConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	logger.Debug("logger debug enabled")

	// Initialize infrastructure components

	// Initialize DB
	pool, err := postgres.NewPool(ctx, &postgresConfig, logger)
	if err != nil {
		return fmt.Errorf("failed to init database: %w", err)
	}

	// Initialize repositories
	sessionRepo := postgres.NewSessionRepo(pool)
	messageRepo := postgres.NewMessageRepo(pool)
	userRepo := postgres.NewUserRepo(pool)
	actionLogRepo := postgres.NewActionLogRepo(pool)

	// Initialize session service
	sessionService := session.NewService(sessionRepo)

	// Initialize LLM client
	llmClient := llm.NewClient(llmConfig, logger)

	// Initialize presenter
	presenter, err := apppresenter.NewPresenter(configPath)
	if err != nil {
		return fmt.Errorf("failed to initialize presenter: %w", err)
	}

	// Initialize processor
	processor := appprocessor.NewProcessor(logger)

	// Register business actions (read-only DB queries)
	findBooking := appactions.NewFindBooking(logger)
	processor.Register("find_booking", findBooking)

	findWorkspaceBooking := appactions.NewFindWorkspaceBooking(logger)
	processor.Register("find_workspace_booking", findWorkspaceBooking)

	findPayment := appactions.NewFindPayment(logger)
	processor.Register("find_payment", findPayment)

	findUserAccount := appactions.NewFindUserAccount(logger)
	processor.Register("find_user_account", findUserAccount)

	// Register utility actions
	validateIdentifier := appactions.NewValidateIdentifier(logger)
	processor.Register("validate_identifier", validateIdentifier)

	// Initialize message worker
	msgWorker := appworker.NewMessageWorker(sessionService, processor, presenter, messageRepo, llmClient, logger)

	// Initialize HTTP transport
	router := http.NewRouter(msgWorker, sessionService, sessionRepo, messageRepo, logger, httpConfig)
	httpServer := http.NewServer(httpConfig, logger, router)

	// Initialize application
	app := NewApp(&loggerConfig, &postgresConfig, logger, pool, llmClient, httpServer, msgWorker, sessionRepo, messageRepo, userRepo, actionLogRepo)

	// Start HTTP server (goroutine is managed internally)
	if err := app.HTTP.Run(ctx); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	logger.Info("application started")

	// Graceful shutdown
	if err := GracefulShutdown(ctx, 30*time.Second, logger, app.HTTP, app.DB); err != nil {
		logger.Error("error during shutdown", logger.Err(err))
		return err
	}

	logger.Info("application stopped successfully")
	return nil
}
