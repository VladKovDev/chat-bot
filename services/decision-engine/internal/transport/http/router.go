package http

import (
	"net/http"

	"github.com/VladKovDev/chat-bot/internal/transport/http/handler"
	"github.com/VladKovDev/chat-bot/internal/transport/http/middleware"
	"github.com/VladKovDev/chat-bot/pkg/logger"
	"github.com/go-chi/chi/v5"
)

func NewRouter(
	messageHandler handler.MessageHandler,
	logger logger.Logger,
	cfg Config,
) http.Handler {
	r := chi.NewRouter()

	if cfg.EnableRecovery {
		r.Use(middleware.RecoveryMiddleware(logger))
	}

	r.Use(middleware.RequestIDMiddleware())

	if cfg.EnableLogs {
		r.Use(middleware.LoggingMiddleware(logger))
	}

	if cfg.BodyLimit > 0 {
		r.Use(middleware.BodyLimitMiddleware(cfg.BodyLimit))
	}

	if cfg.Timeout > 0 {
		r.Use(middleware.TimeoutMiddleware(cfg.Timeout))
	}

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Decision engine endpoints
	h := handler.NewHandler(messageHandler, logger)
	r.Post("/decide", h.Decide)

	// LLM configuration endpoint
	r.Get("/config_llm", h.ConfigLLM)

	return r
}
