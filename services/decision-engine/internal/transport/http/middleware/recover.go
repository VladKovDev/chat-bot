package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/VladKovDev/chat-bot/pkg/logger"
)

func RecoveryMiddleware(logger logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					errMsg := fmt.Sprintf("%v", err)
					logger.Error("panic recovered",
						logger.String("panic", errMsg),
						logger.String("stack", string(debug.Stack())),
						logger.String("method", r.Method),
						logger.String("url", r.URL.String()))
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
