package llm

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/VladKovDev/chat-bot/internal/apperror"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

func TestClientRedactsUpstreamErrorBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"stack":"trace","prompt":"secret","sql":"SELECT * FROM messages"}`, http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:       server.URL,
		Timeout:       time.Second,
		CBMaxRequests: 1,
		CBInterval:    time.Second,
		CBMaxFailures: 1,
		CBTimeout:     time.Second,
	}, logger.Noop())

	_, err := client.Decide(context.Background(), map[string]string{"text": "hello"})
	if err == nil {
		t.Fatal("expected error")
	}
	for _, forbidden := range []string{"secret", "SELECT", "stack", "prompt"} {
		if strings.Contains(err.Error(), forbidden) {
			t.Fatalf("error leaked upstream body fragment %q: %v", forbidden, err)
		}
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("error type = %T, want *apperror.Error", err)
	}
	if appErr.Code != apperror.CodeProviderUnavailable {
		t.Fatalf("code = %q, want %q", appErr.Code, apperror.CodeProviderUnavailable)
	}
}
