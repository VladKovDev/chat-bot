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

func TestClientStartsBrowserSessionAndSendsSessionIdentity(t *testing.T) {
	t.Parallel()

	var sessionRequest dto.SessionRequest
	var messageRequest dto.DecisionEngineRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/sessions":
			if err := json.NewDecoder(r.Body).Decode(&sessionRequest); err != nil {
				t.Fatalf("decode session request: %v", err)
			}
			json.NewEncoder(w).Encode(dto.SessionResponse{
				SessionID: "session-a",
				Channel:   WebsiteChannel,
				ClientID:  sessionRequest.ClientID,
				Resumed:   true,
				Success:   true,
			})
		case "/decide":
			if err := json.NewDecoder(r.Body).Decode(&messageRequest); err != nil {
				t.Fatalf("decode message request: %v", err)
			}
			json.NewEncoder(w).Encode(dto.DecisionEngineResponse{
				Text:      "ok",
				SessionID: messageRequest.SessionID,
				Channel:   messageRequest.Channel,
				ClientID:  messageRequest.ClientID,
				Success:   true,
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

	messageResp, err := c.SendMessage(context.Background(), "hello", sessionResp.SessionID, "browser-a")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	if messageResp.SessionID != "session-a" {
		t.Fatalf("message response session_id = %q", messageResp.SessionID)
	}

	if sessionRequest.Channel != WebsiteChannel || sessionRequest.ClientID != "browser-a" {
		t.Fatalf("session request identity = %+v", sessionRequest)
	}
	if messageRequest.Channel != WebsiteChannel || messageRequest.ClientID != "browser-a" || messageRequest.SessionID != "session-a" {
		t.Fatalf("message request identity = %+v", messageRequest)
	}
}

func TestClientPreservesSafePublicErrorAndDoesNotExposeRawBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(dto.DecisionEngineResponse{
			Success: false,
			Error: &dto.PublicError{
				Code:      "provider_unavailable",
				Message:   "Не удалось проверить данные. Попробуйте позже или подключим оператора.",
				RequestID: "req-42",
			},
		})
	}))
	defer server.Close()

	c := NewClient(config.DecisionEngine{URL: server.URL}, testLogger{})
	resp, err := c.SendMessage(context.Background(), "raw user text", "session-a", "browser-a")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, forbidden := range []string{"raw user text", "SELECT", "prompt"} {
		if strings.Contains(err.Error(), forbidden) {
			t.Fatalf("client error leaked %q: %v", forbidden, err)
		}
	}
	if resp.Error == nil {
		t.Fatalf("expected public error in response: %+v", resp)
	}
	if resp.Error.Code != "provider_unavailable" || resp.Error.RequestID != "req-42" {
		t.Fatalf("public error = %+v", resp.Error)
	}
}

type testLogger struct{}

func (testLogger) Debug(string, ...logger.Field) {}
func (testLogger) Info(string, ...logger.Field)  {}
func (testLogger) Warn(string, ...logger.Field)  {}
func (testLogger) Error(string, ...logger.Field) {}
