package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/VladKovDev/web-adapter/internal/config"
	"github.com/VladKovDev/web-adapter/internal/dto"
	"github.com/VladKovDev/web-adapter/pkg/logger"
)

func TestClientUsesVersionedDecisionEngineEndpoints(t *testing.T) {
	t.Parallel()

	var sessionRequest dto.SessionRequest
	var messageRequest dto.DecisionEngineRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/sessions":
			if err := json.NewDecoder(r.Body).Decode(&sessionRequest); err != nil {
				t.Fatalf("decode session request: %v", err)
			}
			json.NewEncoder(w).Encode(dto.SessionResponse{
				SessionID: "session-a",
				UserID:    "user-a",
				Mode:      "standard",
				Resumed:   true,
			})
		case "/api/v1/messages":
			if err := json.NewDecoder(r.Body).Decode(&messageRequest); err != nil {
				t.Fatalf("decode message request: %v", err)
			}
			json.NewEncoder(w).Encode(dto.DecisionEngineResponse{
				SessionID:     messageRequest.SessionID,
				UserMessageID: messageRequest.EventID,
				BotMessageID:  "bot-message-1",
				Mode:          "standard",
				Text:          "ok",
				CorrelationID: "req-1",
				Timestamp:     "2026-05-10T12:00:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := NewClient(config.DecisionEngine{URL: server.URL}, testLogger{})
	sessionResp, err := c.StartSession(context.Background(), "browser-a")
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if !sessionResp.Resumed || sessionResp.SessionID != "session-a" {
		t.Fatalf("session response = %+v", sessionResp)
	}

	messageResp, err := c.SendMessage(context.Background(), "hello", sessionResp.SessionID, "browser-a", "event-1")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	if messageResp.BotMessageID != "bot-message-1" {
		t.Fatalf("message response = %+v", messageResp)
	}

	if sessionRequest.Channel != WebsiteChannel || sessionRequest.ClientID != "browser-a" {
		t.Fatalf("session request identity = %+v", sessionRequest)
	}
	if messageRequest.Type != "user_message" || messageRequest.EventID != "event-1" {
		t.Fatalf("message request = %+v", messageRequest)
	}
}

func TestClientPreservesStablePublicErrorShape(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(dto.ErrorEnvelope{
			Error: dto.PublicError{
				Code:      "provider_unavailable",
				Message:   "Не удалось проверить данные. Попробуйте позже или подключим оператора.",
				RequestID: "req-42",
			},
		})
	}))
	defer server.Close()

	c := NewClient(config.DecisionEngine{URL: server.URL}, testLogger{})
	_, err := c.SendMessage(context.Background(), "raw user text", "session-a", "browser-a", "event-1")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, forbidden := range []string{"raw user text", "SELECT", "prompt"} {
		if strings.Contains(err.Error(), forbidden) {
			t.Fatalf("client error leaked %q: %v", forbidden, err)
		}
	}
	for _, required := range []string{"provider_unavailable", "req-42"} {
		if !strings.Contains(err.Error(), required) {
			t.Fatalf("client error missing %q: %v", required, err)
		}
	}
}

type testLogger struct{}

func (testLogger) Debug(string, ...logger.Field) {}
func (testLogger) Info(string, ...logger.Field)  {}
func (testLogger) Warn(string, ...logger.Field)  {}
func (testLogger) Error(string, ...logger.Field) {}
