package app

import (
	"context"
	"fmt"
	"os"
	"time"

	appactions "github.com/VladKovDev/chat-bot/internal/app/actions"
	appdecision "github.com/VladKovDev/chat-bot/internal/app/decision"
	appdialogreset "github.com/VladKovDev/chat-bot/internal/app/dialogreset"
	appoperator "github.com/VladKovDev/chat-bot/internal/app/operator"
	apppresenter "github.com/VladKovDev/chat-bot/internal/app/presenter"
	appprocessor "github.com/VladKovDev/chat-bot/internal/app/processor"
	appseed "github.com/VladKovDev/chat-bot/internal/app/seed"
	appworker "github.com/VladKovDev/chat-bot/internal/app/worker"
	"github.com/VladKovDev/chat-bot/internal/config"
	loggerCfg "github.com/VladKovDev/chat-bot/internal/config/logger"
	nlpCfg "github.com/VladKovDev/chat-bot/internal/config/nlp"
	postgresCfg "github.com/VladKovDev/chat-bot/internal/config/postgres"
	transportCfg "github.com/VladKovDev/chat-bot/internal/config/transport"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/message"
	operatorDomain "github.com/VladKovDev/chat-bot/internal/domain/operator"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/user"
	infranlp "github.com/VladKovDev/chat-bot/internal/infrastructure/nlp"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository/postgres"
	"github.com/VladKovDev/chat-bot/internal/transport/http"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

type App struct {
	LoggerConfig   *logger.Config
	PostgresConfig *postgres.Config

	Logger logger.Logger
	DB     *postgres.Pool
	HTTP   *http.Server

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

	httpConfig, err := transportCfg.LoadConfig(viper)
	if err != nil {
		return fmt.Errorf("failed to load http config: %w", err)
	}
	nlpConfig, err := nlpCfg.LoadConfig(viper)
	if err != nil {
		return fmt.Errorf("failed to load nlp config: %w", err)
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
	operatorRepo := postgres.NewOperatorRepo(pool)
	dialogResetRepo := postgres.NewDialogResetRepo(pool)
	messagePersistence := postgres.NewMessagePersistence(pool)
	semanticCatalogRepo := postgres.NewSemanticCatalogRepository(pool.Pool)

	// Initialize session service
	sessionService := session.NewService(sessionRepo)

	// Initialize presenter
	presenter, err := apppresenter.NewPresenter(configPath)
	if err != nil {
		return fmt.Errorf("failed to initialize presenter: %w", err)
	}
	intentCatalog, err := apppresenter.LoadIntentCatalog(configPath)
	if err != nil {
		return fmt.Errorf("failed to load intent catalog: %w", err)
	}
	presenterValidator := apppresenter.NewValidator(presenter.GetAll(), logger)
	if err := presenterValidator.Validate(); err != nil {
		return fmt.Errorf("failed to validate response config: %w", err)
	}
	if err := presenterValidator.ValidateCatalog(intentCatalog); err != nil {
		return fmt.Errorf("failed to validate intent catalog: %w", err)
	}
	dataset, err := appseed.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load seed dataset: %w", err)
	}
	if err := dataset.ValidateCatalog(intentCatalog); err != nil {
		return fmt.Errorf("failed to validate seed dataset: %w", err)
	}
	for _, fixture := range dataset.Operators.Items {
		if _, err := operatorRepo.UpsertOperator(ctx, operatorDomain.Account{
			OperatorID:  fixture.OperatorID,
			FixtureID:   fixture.ID,
			DisplayName: fixture.Name,
			Status:      fixture.Status,
		}); err != nil {
			return fmt.Errorf("failed to persist demo operator %s: %w", fixture.OperatorID, err)
		}
	}
	if err := semanticCatalogRepo.SeedDemoData(ctx, dataset); err != nil {
		return fmt.Errorf("failed to persist demo seed data: %w", err)
	}
	embedder, err := infranlp.NewEmbedderClient(nlpConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize nlp embedder: %w", err)
	}
	semanticMatcher, err := appdecision.NewSemanticIntentMatcher(
		embedder,
		semanticCatalogRepo,
		appdecision.SemanticMatcherConfig{TopK: 3, Locale: "ru"},
	)
	if err != nil {
		return fmt.Errorf("failed to initialize semantic matcher: %w", err)
	}
	if err := appseed.SeedSemanticCatalog(ctx, intentCatalog, dataset, semanticCatalogRepo, embedder); err != nil {
		logger.Error("semantic catalog seed failed; continuing with exact-command and low-confidence fallback",
			logger.Err(err))
	}
	decisionService, err := appdecision.NewService(intentCatalog, semanticMatcher, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize decision service: %w", err)
	}
	knowledgeRetriever, err := appdecision.NewKnowledgeRetriever(
		embedder,
		semanticCatalogRepo,
		appdecision.KnowledgeRetrieverConfig{TopK: 3},
	)
	if err != nil {
		return fmt.Errorf("failed to initialize knowledge retriever: %w", err)
	}
	decisionService.SetKnowledgeSearcher(knowledgeRetriever)

	operatorService := appoperator.NewService(operatorRepo, sessionRepo)
	dialogResetService := appdialogreset.NewService(dialogResetRepo, logger)

	// Initialize processor
	processor := appprocessor.NewProcessor(logger)

	// Register business actions (read-only DB queries)
	findBooking := appactions.NewFindBooking(logger, dataset)
	processor.Register("find_booking", findBooking)

	findWorkspaceBooking := appactions.NewFindWorkspaceBooking(logger, dataset)
	processor.Register("find_workspace_booking", findWorkspaceBooking)

	findPayment := appactions.NewFindPayment(logger, dataset)
	processor.Register("find_payment", findPayment)

	findUserAccount := appactions.NewFindUserAccount(logger, dataset)
	processor.Register("find_user_account", findUserAccount)

	// Register utility actions
	validateIdentifier := appactions.NewValidateIdentifier(logger)
	processor.Register("validate_identifier", validateIdentifier)
	processor.Register(action.ActionEscalateToOperator, appactions.NewEscalateToOperator())

	// Initialize message worker
	msgWorker := appworker.NewMessageWorker(sessionService, decisionService, processor, presenter, messagePersistence, logger, operatorService)

	// Initialize HTTP transport
	readiness := NewReadinessProvider(pool, nlpConfig)
	router := http.NewRouter(msgWorker, sessionService, sessionRepo, messageRepo, logger, httpConfig, readiness, dialogResetService, operatorService)
	httpServer := http.NewServer(httpConfig, logger, router)

	// Initialize application
	app := NewApp(&loggerConfig, &postgresConfig, logger, pool, httpServer, msgWorker, sessionRepo, messageRepo, userRepo, actionLogRepo)

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
