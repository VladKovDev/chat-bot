package middleware

import (
	"net/http"
	"time"

	"github.com/VladKovDev/chat-bot/pkg/logger"
	"github.com/go-chi/chi/v5/middleware"
)

func LoggingMiddleware(logger logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			logger.Info("HTTP request",
				logger.String("method", r.Method),
				logger.String("url", r.URL.String()),
				logger.Int("status", ww.Status()),
				logger.Int("bytes", ww.BytesWritten()),
				logger.Any("duration", time.Since(start)),
				logger.String("request_id", r.Header.Get("X-Request-ID")),
			)
		})
	}
}
