package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			RequestID := r.Header.Get("X-Request-ID")
			if RequestID == "" {
				RequestID = uuid.New().String()
			}

			// Add the Request ID to the request context
			ctx := context.WithValue(r.Context(), "RequestID", RequestID)
			r = r.WithContext(ctx)

			// Set the Request ID in the response header
			w.Header().Set("X-Request-ID", RequestID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}