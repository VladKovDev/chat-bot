package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/VladKovDev/web-adapter/internal/client"
	"github.com/VladKovDev/web-adapter/internal/config"
	"github.com/VladKovDev/web-adapter/internal/websocket"
	"github.com/VladKovDev/web-adapter/pkg/logger"
	"github.com/spf13/viper"
)

// App represents the application
type App struct {
	config  config.Config
	Logger  logger.Logger
	client  *client.Client
	handler *websocket.Handler
	server  *websocket.Server
}

// New creates a new application
func New(v *viper.Viper) (*App, error) {
	// Load configuration
	cfg, err := config.LoadConfig(v)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Create logger
	log, err := logger.New(logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	log.Info("initializing application",
		logger.String("decision_engine_url", cfg.DecisionEngine.URL),
		logger.String("server_address", cfg.Server.Address),
	)

	// Create decision engine client
	client := client.NewClient(cfg.DecisionEngine, log)

	// Create WebSocket handler
	handler := websocket.NewHandler(client, cfg.Server, log)

	// Create server
	srv := websocket.NewServer(cfg.Server, handler, log)

	return &App{
		config:  cfg,
		Logger:  log,
		client:  client,
		handler: handler,
		server:  srv,
	}, nil
}

// Run starts the application
func (a *App) Run() error {
	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := a.server.Start(); err != nil {
			errCh <- err
		}
	}()

	a.Logger.Info("application started",
		logger.String("address", a.config.Server.Address),
		logger.String("decision_engine", a.config.DecisionEngine.URL),
	)

	// Wait for shutdown signal or error
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigCh:
		a.Logger.Info("received shutdown signal")
	case err := <-errCh:
		a.Logger.Error("server error", logger.Err(err))
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the application
func (a *App) Shutdown() error {
	a.Logger.Info("shutting down application")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
	if err := a.server.Shutdown(ctx); err != nil {
		a.Logger.Error("failed to shutdown server", logger.Err(err))
		return err
	}

	a.Logger.Info("application shutdown complete")
	return nil
}
