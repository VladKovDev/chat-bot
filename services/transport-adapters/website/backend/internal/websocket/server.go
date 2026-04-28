package websocket

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/VladKovDev/web-adapter/internal/config"
	"github.com/VladKovDev/web-adapter/pkg/logger"
)

// Server represents the WebSocket server
type Server struct {
	httpServer *http.Server
	handler    *Handler
	logger     logger.Logger
}

// NewServer creates a new WebSocket server
func NewServer(cfg config.Server, handler *Handler, log logger.Logger) *Server {
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", handler.HandleConnection)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Serve static files (frontend)
	mux.Handle("/", http.FileServer(http.Dir("./frontend")))

	httpServer := &http.Server{
		Addr:           cfg.Address,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	return &Server{
		httpServer: httpServer,
		handler:    handler,
		logger:     log,
	}
}

// Start starts the WebSocket server
func (s *Server) Start() error {
	s.logger.Info("starting websocket server",
		logger.String("address", s.httpServer.Addr),
	)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down websocket server")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	return nil
}
