package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/VladKovDev/chat-bot/pkg/logger"
)

type Shutdowner interface {
	Shutdown(ctx context.Context) error
}

func GracefulShutdown(ctx context.Context, timeout time.Duration, logger logger.Logger, shutdowners ...Shutdowner) error {
	// Create context for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create channel for signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal
	sig := <-sigChan
	logger.Info("received shutdown signal", logger.String("signal", sig.String()))

	// Shutdown all components
	var err error
	for _, shutdowner := range shutdowners {
		if shutdownErr := shutdowner.Shutdown(shutdownCtx); shutdownErr != nil {
			logger.Error("error during shutdown", logger.Err(shutdownErr))
			if err == nil {
				err = shutdownErr
			}
		}
	}

	return err
}