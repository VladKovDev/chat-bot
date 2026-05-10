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
			messageRequest = dto.DecisionEngineRequest{}
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

	_, err = c.SendQuickReply(context.Background(), dto.QuickReply{
		ID:     "renamed-menu",
		Label:  "Changed label",
		Action: "select_intent",
		Payload: map[string]any{
			"intent": "return_to_menu",
		},
	}, sessionResp.SessionID, "browser-a", "event-quick-reply")
	if err != nil {
		t.Fatalf("send quick reply: %v", err)
	}
	if messageRequest.Type != "quick_reply.selected" || messageRequest.QuickReply == nil {
		t.Fatalf("quick reply request = %+v", messageRequest)
	}
	if messageRequest.QuickReply.ID != "renamed-menu" || messageRequest.Text != "" {
		t.Fatalf("quick reply request = %+v", messageRequest)
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

func TestClientUsesOperatorEndpoints(t *testing.T) {
	t.Parallel()

	var accepted dto.OperatorQueueActionRequest
	var operatorMessage dto.OperatorMessageRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/operator/queue":
			if r.URL.Query().Get("status") != "waiting" {
				t.Fatalf("queue status query = %q, want waiting", r.URL.RawQuery)
			}
			json.NewEncoder(w).Encode(dto.OperatorQueueResponse{
				Items: []dto.OperatorQueueItem{{
					HandoffID:     "handoff-a",
					SessionID:     "session-a",
					Status:        "waiting",
					Reason:        "manual_request",
					FallbackCount: 2,
					CreatedAt:     "2026-05-10T12:00:00Z",
					Preview:       "Нужен оператор",
				}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/operator/queue/handoff-a/accept":
			if err := json.NewDecoder(r.Body).Decode(&accepted); err != nil {
				t.Fatalf("decode accept request: %v", err)
			}
			json.NewEncoder(w).Encode(dto.OperatorQueueActionResponse{
				Handoff: dto.Handoff{HandoffID: "handoff-a", SessionID: "session-a", Status: "accepted"},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/operator/sessions/session-a/messages":
			if err := json.NewDecoder(r.Body).Decode(&operatorMessage); err != nil {
				t.Fatalf("decode operator message request: %v", err)
			}
			json.NewEncoder(w).Encode(dto.OperatorMessageResponse{
				SessionID:  "session-a",
				MessageID:  "message-operator-a",
				OperatorID: operatorMessage.OperatorID,
				Text:       operatorMessage.Text,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := NewClient(config.DecisionEngine{URL: server.URL}, testLogger{})
	queue, err := c.GetOperatorQueue(context.Background(), "waiting")
	if err != nil {
		t.Fatalf("get operator queue: %v", err)
	}
	if len(queue.Items) != 1 || queue.Items[0].FallbackCount != 2 {
		t.Fatalf("operator queue = %+v", queue)
	}

	if _, err := c.AcceptHandoff(context.Background(), "handoff-a", "operator-1"); err != nil {
		t.Fatalf("accept handoff: %v", err)
	}
	if accepted.OperatorID != "operator-1" {
		t.Fatalf("accept operator_id = %q", accepted.OperatorID)
	}

	resp, err := c.SendOperatorMessage(context.Background(), "session-a", "operator-1", "Здравствуйте")
	if err != nil {
		t.Fatalf("send operator message: %v", err)
	}
	if resp.MessageID != "message-operator-a" || operatorMessage.Text != "Здравствуйте" {
		t.Fatalf("operator message resp=%+v request=%+v", resp, operatorMessage)
	}
}

type testLogger struct{}

func (testLogger) Debug(string, ...logger.Field) {}
func (testLogger) Info(string, ...logger.Field)  {}
func (testLogger) Warn(string, ...logger.Field)  {}
func (testLogger) Error(string, ...logger.Field) {}
