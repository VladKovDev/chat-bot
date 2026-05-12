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
	sessionService handler.SessionService,
	sessionRepo handler.SessionRepository,
	messageRepo handler.MessageRepository,
	logger logger.Logger,
	cfg Config,
	readiness handler.ReadinessProvider,
	resetter handler.DialogResetter,
	operatorQueue ...handler.OperatorQueueService,
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

	h := handler.NewHandler(messageHandler, sessionService, sessionRepo, messageRepo, logger, operatorQueue...)
	h.SetReadiness(readiness)
	h.SetDialogResetter(resetter, cfg.AdminResetToken)

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/health", h.Health)
		api.Get("/ready", h.Ready)
		api.Post("/sessions", h.StartSession)
		api.Post("/messages", h.Message)
		api.Get("/sessions/{session_id}/messages", h.SessionMessages)
		api.Get("/domain/schema", h.DomainSchema)
		api.Post("/admin/sessions/{session_id}/reset", h.ResetSession)
		api.Post("/operator/queue/{session_id}/request", h.RequestOperator)
		api.Get("/operator/queue", h.OperatorQueue)
		api.Post("/operator/queue/{handoff_id}/accept", h.AcceptOperatorQueue)
		api.Post("/operator/sessions/{session_id}/messages", h.OperatorMessage)
		api.Post("/operator/queue/{handoff_id}/close", h.CloseOperatorQueue)
	})

	return r
}
