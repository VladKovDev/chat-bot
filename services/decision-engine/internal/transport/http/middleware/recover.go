package middleware

import (
	"net/http"

	"github.com/VladKovDev/chat-bot/internal/apperror"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

func RecoveryMiddleware(logger logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID := RequestIDFromRequest(r)
					logger.Error("panic recovered",
						logger.String("request_id", requestID),
						logger.String("error_code", string(apperror.CodeInternal)),
						logger.String("method", r.Method),
						logger.String("url", r.URL.String()))
					apperror.WriteJSON(
						w,
						http.StatusInternalServerError,
						apperror.NewPublic(apperror.CodeInternal, requestID),
					)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
