package app

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/VladKovDev/chat-bot/internal/app/actions"
	"github.com/VladKovDev/chat-bot/internal/app/presenter"
	"github.com/VladKovDev/chat-bot/internal/app/processor"
	"github.com/VladKovDev/chat-bot/internal/app/transition"
	appworker "github.com/VladKovDev/chat-bot/internal/app/worker"
	"github.com/VladKovDev/chat-bot/internal/config"
	lemmatizerCfg "github.com/VladKovDev/chat-bot/internal/config/lemmatizer"
	loggerCfg "github.com/VladKovDev/chat-bot/internal/config/logger"
	postgresCfg "github.com/VladKovDev/chat-bot/internal/config/postgres"
	transportCfg "github.com/VladKovDev/chat-bot/internal/config/transport"
	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
	"github.com/VladKovDev/chat-bot/internal/domain/intent"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/lemmatizer"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/nlp"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/nlp/normalization"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/nlp/rule_based"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository/postgres"
	"github.com/VladKovDev/chat-bot/internal/transport/http"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

type App struct {
	LoggerConfig   *logger.Config
	PostgresConfig *postgres.Config

	Logger logger.Logger
	DB     *postgres.Pool
	NLP    *nlp.Classifier
	HTTP   *http.Server

	ConversationRepo conversation.Repository
	Worker           *appworker.MessageWorker
}

func NewApp(
	loggerConfig *logger.Config,
	postgresConfig *postgres.Config,
	logger logger.Logger,
	db *postgres.Pool,
	nlp *nlp.Classifier,
	httpServer *http.Server,
	worker *appworker.MessageWorker,
	conversationRepo conversation.Repository,
) *App {

	return &App{
		LoggerConfig:   loggerConfig,
		PostgresConfig: postgresConfig,
		Logger:         logger,
		DB:             db,
		NLP:            nlp,
		HTTP:           httpServer,
		Worker:         worker,
		ConversationRepo: conversationRepo,
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

	lemmatizerConfig, err := lemmatizerCfg.LoadConfig(viper)
	if err != nil {
		return fmt.Errorf("failed to load lemmatizer config: %w", err)
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

	// Initialize lemmatizer
	lemmatizerClient := lemmatizer.NewClient(lemmatizerConfig, logger)

	// Initialize NLP normalizer pipeline
	normalizer := normalization.NewPipeline(lemmatizerClient, 5*time.Second, logger)

	// Initialize rule-based classifier
	ruleBasedConfig, err := rule_based.LoadRules(configPath + "/rules.json")
	if err != nil {
		return fmt.Errorf("failed to load rule-based config: %w", err)
	}
	ruleBasedClassifier, err := rule_based.NewRuleBased(ruleBasedConfig, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize rule-based classifier: %w", err)
	}

	// Initialize Intent → Event mapping
	intentToEvent := map[intent.Intent]conversation.Event{
		intent.IntentGreeting:           conversation.EventGreeting,
		intent.IntentCategorySelect:     conversation.EventCategorySelected,
		intent.IntentRequestOperator:    conversation.EventRequestOperator,
		intent.IntentResetConversation:  conversation.EventResetConversation,
		intent.IntentResolved:           conversation.EventResolved,
		intent.IntentNotResolved:        conversation.EventNotResolved,
		intent.IntentOperatorClosed:     conversation.EventOperatorClosed,
		intent.IntentUnknown:            conversation.EventMessageReceived, // default
	}

	// Initialize Event adapter
	eventAdapter := nlp.NewEventAdapter(intentToEvent)

	// Initialize NLP classifier with Event adapter
	nlpClassifier := nlp.NewClassifier(ruleBasedClassifier, normalizer, eventAdapter, logger)

	// Load transition configuration
	transitionConfig, err := transition.LoadConfig(configPath + "/transitions.json")
	if err != nil {
		return fmt.Errorf("failed to load transition config: %w", err)
	}

	// Build transition maps
	transitions, globalEvents := transition.BuildTransitionMaps(transitionConfig)

	// Initialize transition engine
	transitionEngine := transition.NewEngine(transitions, globalEvents, logger)

	// Initialize conversation repository
	conversationRepo := postgres.NewConversationRepo(pool)

	// Initialize conversation service
	conversationService := conversation.NewService(conversationRepo)

	// Initialize presenter
	responseLoader, err := presenter.NewLoader(configPath)
	if err != nil {
		return fmt.Errorf("failed to initialize response loader: %w", err)
	}
	presenter := presenter.NewPresenter(responseLoader)

	// Initialize processor and register actions
	proc := processor.NewProcessor(logger)
	proc.Register("escalate_operator", actions.NewEscalateOperatorAction(logger))
	proc.Register("reset_conversation", actions.NewResetConversationAction(conversationRepo, logger))

	// Initialize message worker
	msgWorker := appworker.NewMessageWorker(
		conversationService,
		transitionEngine,
		proc,
		presenter,
		nlpClassifier,
		logger,
	)

	// Initialize HTTP transport
	router := http.NewRouter(msgWorker, logger, httpConfig)
	httpServer := http.NewServer(httpConfig, logger, router)

	// Initialize application
	app := NewApp(&loggerConfig, &postgresConfig, logger, pool, nlpClassifier, httpServer, msgWorker, conversationRepo)

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