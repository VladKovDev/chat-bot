package http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/VladKovDev/chat-bot/pkg/logger"
)

type Server struct {
	httpServer *http.Server
	logger     logger.Logger
	cfg        Config
	wg         sync.WaitGroup
}

func NewServer(cfg Config, logger logger.Logger, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:              cfg.Address,
			Handler:           handler,
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
			ReadHeaderTimeout: cfg.ReadHeadTimeout,
			MaxHeaderBytes:    cfg.MaxHeaderBytes,
		},
		logger: logger,
		cfg:    cfg,
	}
}

func (s *Server) Run(ctx context.Context) error {
	s.logger.Info("Starting HTTP server", s.logger.String("address", s.cfg.Address))

	ln, err := net.Listen("tcp", s.cfg.Address)
	if err != nil {
		return err
	}

	// Track goroutine for graceful shutdown
	s.wg.Add(1)

	// Run server in goroutine
	go func() {
		defer s.wg.Done()

		// Serve will return http.ErrServerClosed when Shutdown is called
		if err := s.httpServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("HTTP server error", s.logger.Err(err))
		}
	}()

	s.logger.Info("HTTP server started successfully")
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			s.logger.Warn("HTTP server shutdown timeout, forcing close")
			// Force close if timeout exceeded
			s.httpServer.Close()
		} else if !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("HTTP server shutdown error", s.logger.Err(err))
			return err
		}
	}

	// Wait for server goroutine to finish
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("HTTP server shutdown completed successfully")
		return nil
	case <-time.After(35 * time.Second):
		s.logger.Warn("HTTP server shutdown wait timeout")
		return ctx.Err()
	}
}