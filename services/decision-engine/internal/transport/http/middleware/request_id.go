package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type requestIDContextKey struct{}

func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}

			ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
			r = r.WithContext(ctx)

			w.Header().Set("X-Request-ID", requestID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func RequestIDFromRequest(r *http.Request) string {
	if requestID := RequestIDFromContext(r.Context()); requestID != "" {
		return requestID
	}
	return r.Header.Get("X-Request-ID")
}
