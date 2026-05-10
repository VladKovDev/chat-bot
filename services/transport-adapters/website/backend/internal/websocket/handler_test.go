package websocket

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/VladKovDev/web-adapter/internal/client"
	"github.com/VladKovDev/web-adapter/internal/config"
	"github.com/VladKovDev/web-adapter/internal/dto"
	"github.com/VladKovDev/web-adapter/pkg/logger"
	"github.com/gorilla/websocket"
)

var websocketFixedNow = time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

const testAllowedOrigin = "https://chat.example.test"

func TestWebSocketContractDocumentListsTypedEvents(t *testing.T) {
	t.Parallel()

	doc := loadWebSocketContract(t)

	clientEvents, ok := doc["client_events"].(map[string]any)
	if !ok {
		t.Fatalf("client_events is not an object: %#v", doc["client_events"])
	}
	serverEvents, ok := doc["server_events"].(map[string]any)
	if !ok {
		t.Fatalf("server_events is not an object: %#v", doc["server_events"])
	}

	for _, eventType := range []string{
		dto.EventSessionStart,
		dto.EventMessageUser,
		dto.EventQuickReplySelected,
		dto.EventOperatorClose,
	} {
		if _, ok := clientEvents[eventType]; !ok {
			t.Fatalf("client event %q missing from contract document", eventType)
		}
	}

	for _, eventType := range []string{
		dto.EventSessionStarted,
		dto.EventMessageBot,
		dto.EventMessageOperator,
		dto.EventHandoffQueued,
		dto.EventHandoffAccepted,
		dto.EventHandoffClosed,
		dto.EventError,
	} {
		if _, ok := serverEvents[eventType]; !ok {
			t.Fatalf("server event %q missing from contract document", eventType)
		}
	}

	rawDoc, err := os.ReadFile(webSocketContractPath())
	if err != nil {
		t.Fatalf("read websocket contract: %v", err)
	}
	text := string(rawDoc)
	for _, forbidden := range []string{`"type": "response"`, `"type": "session"`} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("websocket contract still contains legacy ambiguous event name %q", forbidden)
		}
	}
}

func TestHandleConnectionEmitsTypedV1Events(t *testing.T) {
	t.Parallel()

	var capturedMessage dto.DecisionEngineRequest

	decisionEngine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sessions":
			_ = json.NewEncoder(w).Encode(dto.SessionResponse{
				SessionID: "11111111-1111-1111-1111-111111111111",
				UserID:    "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				Mode:      "standard",
				Resumed:   false,
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/messages":
			if err := json.NewDecoder(r.Body).Decode(&capturedMessage); err != nil {
				t.Fatalf("decode message request: %v", err)
			}
			_ = json.NewEncoder(w).Encode(dto.DecisionEngineResponse{
				SessionID:     capturedMessage.SessionID,
				UserMessageID: capturedMessage.EventID,
				BotMessageID:  "22222222-2222-2222-2222-222222222222",
				Mode:          "standard",
				Text:          "Готово",
				QuickReplies: []dto.QuickReply{
					{
						ID:     "operator",
						Label:  "Связаться с оператором",
						Action: "request_operator",
						Payload: map[string]any{
							"reason": "manual_request",
						},
					},
				},
				CorrelationID: "req-message-1",
				Timestamp:     websocketFixedNow.Format(time.RFC3339Nano),
			})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/request"):
			_ = json.NewEncoder(w).Encode(dto.OperatorQueueActionResponse{
				Handoff: dto.Handoff{
					HandoffID: "11111111-1111-1111-1111-111111111111",
					SessionID: "11111111-1111-1111-1111-111111111111",
					Status:    "waiting",
					Reason:    "manual_request",
				},
			})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/close"):
			_ = json.NewEncoder(w).Encode(dto.OperatorQueueActionResponse{
				Handoff: dto.Handoff{
					HandoffID: "11111111-1111-1111-1111-111111111111",
					SessionID: "11111111-1111-1111-1111-111111111111",
					Status:    "closed",
					Reason:    "manual_request",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer decisionEngine.Close()

	wsHandler := NewHandler(
		client.NewClient(config.DecisionEngine{
			URL:     decisionEngine.URL,
			Timeout: time.Second,
		}, noopLogger{}),
		config.Server{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			AllowedOrigins:  []string{testAllowedOrigin},
		},
		noopLogger{},
	)
	wsHandler.now = func() time.Time { return websocketFixedNow }

	server := httptest.NewServer(http.HandlerFunc(wsHandler.HandleConnection))
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), testWebSocketHeaders())
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(dto.ClientEvent{
		Type:          dto.EventSessionStart,
		EventID:       "evt-session-start",
		CorrelationID: "corr-session-start",
		Timestamp:     websocketFixedNow.Format(time.RFC3339Nano),
		ClientID:      "browser-a",
	}); err != nil {
		t.Fatalf("write session.start event: %v", err)
	}

	started := readEvent[dto.SessionStartedEvent](t, conn)
	if started.Type != dto.EventSessionStarted {
		t.Fatalf("session.started type = %q", started.Type)
	}
	if started.SessionID == "" {
		t.Fatalf("session.started session_id is empty: %+v", started)
	}
	if started.EventID != "evt-session-start" {
		t.Fatalf("session.started event_id = %q, want evt-session-start", started.EventID)
	}
	if started.CorrelationID != "corr-session-start" {
		t.Fatalf("session.started correlation_id = %q, want corr-session-start", started.CorrelationID)
	}

	if err := conn.WriteJSON(dto.ClientEvent{
		Type:          dto.EventMessageUser,
		SessionID:     started.SessionID,
		EventID:       "evt-user-message",
		CorrelationID: "corr-user-message",
		Timestamp:     websocketFixedNow.Format(time.RFC3339Nano),
		Text:          "Помогите",
	}); err != nil {
		t.Fatalf("write message.user event: %v", err)
	}

	botMessage := readEvent[dto.MessageBotEvent](t, conn)
	if botMessage.Type != dto.EventMessageBot {
		t.Fatalf("message.bot type = %q", botMessage.Type)
	}
	if botMessage.SessionID != started.SessionID {
		t.Fatalf("message.bot session_id = %q, want %q", botMessage.SessionID, started.SessionID)
	}
	if botMessage.MessageID != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("message.bot message_id = %q", botMessage.MessageID)
	}
	if botMessage.CorrelationID != "req-message-1" {
		t.Fatalf("message.bot correlation_id = %q, want req-message-1", botMessage.CorrelationID)
	}
	if len(botMessage.QuickReplies) != 1 || botMessage.QuickReplies[0].Action != "request_operator" {
		t.Fatalf("message.bot quick replies = %#v", botMessage.QuickReplies)
	}

	if capturedMessage.Type != "user_message" || capturedMessage.EventID != "evt-user-message" {
		t.Fatalf("captured decision-engine request = %+v", capturedMessage)
	}

	if err := conn.WriteJSON(dto.ClientEvent{
		Type:          dto.EventQuickReplySelected,
		SessionID:     started.SessionID,
		EventID:       "evt-quick-reply",
		CorrelationID: "corr-quick-reply",
		Timestamp:     websocketFixedNow.Format(time.RFC3339Nano),
		QuickReply: &dto.QuickReply{
			ID:     "operator",
			Label:  "Связаться с оператором",
			Action: "request_operator",
		},
	}); err != nil {
		t.Fatalf("write quick_reply.selected event: %v", err)
	}

	queued := readEvent[dto.HandoffEvent](t, conn)
	if queued.Type != dto.EventHandoffQueued {
		t.Fatalf("handoff queued type = %q", queued.Type)
	}
	if queued.Handoff.Status != "waiting" {
		t.Fatalf("handoff queued status = %q, want waiting", queued.Handoff.Status)
	}
	if queued.SessionID != started.SessionID {
		t.Fatalf("handoff queued session_id = %q, want %q", queued.SessionID, started.SessionID)
	}

	if err := conn.WriteJSON(dto.ClientEvent{
		Type:          dto.EventOperatorClose,
		SessionID:     started.SessionID,
		EventID:       "evt-operator-close",
		CorrelationID: "corr-operator-close",
		Timestamp:     websocketFixedNow.Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatalf("write operator.close event: %v", err)
	}

	closed := readEvent[dto.HandoffEvent](t, conn)
	if closed.Type != dto.EventHandoffClosed {
		t.Fatalf("handoff closed type = %q", closed.Type)
	}
	if closed.Handoff.Status != "closed" {
		t.Fatalf("handoff closed status = %q, want closed", closed.Handoff.Status)
	}
}

func TestHandleConnectionForwardsQuickReplySelectedWithoutParsingLabel(t *testing.T) {
	t.Parallel()

	var capturedMessage dto.DecisionEngineRequest

	decisionEngine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sessions":
			_ = json.NewEncoder(w).Encode(dto.SessionResponse{
				SessionID: "11111111-1111-1111-1111-111111111111",
				UserID:    "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				Mode:      "standard",
				Resumed:   false,
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/messages":
			if err := json.NewDecoder(r.Body).Decode(&capturedMessage); err != nil {
				t.Fatalf("decode message request: %v", err)
			}
			_ = json.NewEncoder(w).Encode(dto.DecisionEngineResponse{
				SessionID:     capturedMessage.SessionID,
				UserMessageID: capturedMessage.EventID,
				BotMessageID:  "22222222-2222-2222-2222-222222222222",
				Mode:          "standard",
				Text:          "Меню открыто.",
				CorrelationID: "req-quick-reply",
				Timestamp:     websocketFixedNow.Format(time.RFC3339Nano),
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer decisionEngine.Close()

	conn := dialTestWebSocket(t, decisionEngine.URL)
	defer conn.Close()

	if err := conn.WriteJSON(dto.ClientEvent{
		Type:          dto.EventSessionStart,
		EventID:       "evt-session-start",
		CorrelationID: "corr-session-start",
		Timestamp:     websocketFixedNow.Format(time.RFC3339Nano),
		ClientID:      "browser-a",
	}); err != nil {
		t.Fatalf("write session.start event: %v", err)
	}
	started := readEvent[dto.SessionStartedEvent](t, conn)

	if err := conn.WriteJSON(dto.ClientEvent{
		Type:          dto.EventQuickReplySelected,
		SessionID:     started.SessionID,
		EventID:       "33333333-3333-3333-3333-333333333333",
		CorrelationID: "corr-quick-reply",
		Timestamp:     websocketFixedNow.Format(time.RFC3339Nano),
		QuickReply: &dto.QuickReply{
			ID:     "renamed-menu",
			Label:  `<img src=x onerror="alert(1)">`,
			Action: "select_intent",
			Payload: map[string]any{
				"intent": "return_to_menu",
			},
		},
	}); err != nil {
		t.Fatalf("write quick_reply.selected event: %v", err)
	}

	botMessage := readEvent[dto.MessageBotEvent](t, conn)
	if botMessage.Text != "Меню открыто." {
		t.Fatalf("bot text = %q", botMessage.Text)
	}

	if capturedMessage.Type != dto.EventQuickReplySelected {
		t.Fatalf("captured type = %q, want quick_reply.selected", capturedMessage.Type)
	}
	if capturedMessage.Text == `<img src=x onerror="alert(1)">` {
		t.Fatalf("quick reply label was used as decision text")
	}
	if capturedMessage.QuickReply == nil || capturedMessage.QuickReply.ID != "renamed-menu" {
		t.Fatalf("captured quick reply = %#v", capturedMessage.QuickReply)
	}
	if got := capturedMessage.QuickReply.Payload["intent"]; got != "return_to_menu" {
		t.Fatalf("captured payload intent = %#v", got)
	}
}

func TestHandleConnectionReturnsTypedErrorEventForInvalidPayload(t *testing.T) {
	t.Parallel()

	wsHandler := NewHandler(
		client.NewClient(config.DecisionEngine{
			URL:     "http://127.0.0.1:1",
			Timeout: 100 * time.Millisecond,
		}, noopLogger{}),
		config.Server{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			AllowedOrigins:  []string{testAllowedOrigin},
		},
		noopLogger{},
	)
	wsHandler.now = func() time.Time { return websocketFixedNow }

	server := httptest.NewServer(http.HandlerFunc(wsHandler.HandleConnection))
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), testWebSocketHeaders())
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(dto.ClientEvent{
		Type:          dto.EventMessageUser,
		EventID:       "evt-invalid",
		CorrelationID: "corr-invalid",
		Timestamp:     websocketFixedNow.Format(time.RFC3339Nano),
		Text:          "",
	}); err != nil {
		t.Fatalf("write invalid event: %v", err)
	}

	errorEvent := readEvent[dto.ErrorEvent](t, conn)
	if errorEvent.Type != dto.EventError {
		t.Fatalf("error event type = %q", errorEvent.Type)
	}
	if errorEvent.Error.Code != "invalid_request" {
		t.Fatalf("error code = %q, want invalid_request", errorEvent.Error.Code)
	}
	if errorEvent.CorrelationID != "corr-invalid" {
		t.Fatalf("error correlation_id = %q, want corr-invalid", errorEvent.CorrelationID)
	}
	if _, err := time.Parse(time.RFC3339Nano, errorEvent.Timestamp); err != nil {
		t.Fatalf("error timestamp is not RFC3339: %v", err)
	}
}

func TestHandleConnectionStreamsOperatorEventsFromDecisionEngineRuntime(t *testing.T) {
	t.Parallel()

	type decisionEngineState struct {
		mu            sync.Mutex
		handoffStatus string
		handoffReason string
		messages      []dto.SessionMessageRecord
	}

	state := &decisionEngineState{
		handoffReason: "manual_request",
	}

	decisionEngine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sessions":
			_ = json.NewEncoder(w).Encode(dto.SessionResponse{
				SessionID: "11111111-1111-1111-1111-111111111111",
				UserID:    "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				Mode:      "standard",
				Resumed:   false,
			})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/request"):
			state.mu.Lock()
			state.handoffStatus = "waiting"
			state.mu.Unlock()
			_ = json.NewEncoder(w).Encode(dto.OperatorQueueActionResponse{
				Handoff: dto.Handoff{
					HandoffID: "11111111-1111-1111-1111-111111111111",
					SessionID: "11111111-1111-1111-1111-111111111111",
					Status:    "waiting",
					Reason:    "manual_request",
				},
			})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/operator/queue"):
			status := r.URL.Query().Get("status")
			state.mu.Lock()
			currentStatus := state.handoffStatus
			reason := state.handoffReason
			state.mu.Unlock()

			resp := dto.OperatorQueueResponse{}
			if status == currentStatus && status != "" {
				resp.Items = append(resp.Items, dto.OperatorQueueItem{
					HandoffID: "11111111-1111-1111-1111-111111111111",
					SessionID: "11111111-1111-1111-1111-111111111111",
					Reason:    reason,
					CreatedAt: websocketFixedNow.Add(time.Second).Format(time.RFC3339Nano),
					Preview:   "Оператор подключается",
				})
			}
			_ = json.NewEncoder(w).Encode(resp)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/messages"):
			state.mu.Lock()
			items := append([]dto.SessionMessageRecord(nil), state.messages...)
			state.mu.Unlock()
			_ = json.NewEncoder(w).Encode(dto.SessionMessagesResponse{Items: items})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/accept"):
			state.mu.Lock()
			state.handoffStatus = "accepted"
			state.mu.Unlock()
			_ = json.NewEncoder(w).Encode(dto.OperatorQueueActionResponse{
				Handoff: dto.Handoff{
					HandoffID: "11111111-1111-1111-1111-111111111111",
					SessionID: "11111111-1111-1111-1111-111111111111",
					Status:    "accepted",
					Reason:    "manual_request",
				},
			})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/operator/sessions/") && strings.HasSuffix(r.URL.Path, "/messages"):
			var req struct {
				OperatorID string `json:"operator_id"`
				Text       string `json:"text"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode operator message request: %v", err)
			}
			state.mu.Lock()
			state.messages = append(state.messages, dto.SessionMessageRecord{
				MessageID:  "33333333-3333-3333-3333-333333333333",
				SessionID:  "11111111-1111-1111-1111-111111111111",
				SenderType: "operator",
				Text:       req.Text,
				Timestamp:  websocketFixedNow.Add(2 * time.Second).Format(time.RFC3339Nano),
			})
			state.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer decisionEngine.Close()

	wsHandler := NewHandler(
		client.NewClient(config.DecisionEngine{
			URL:     decisionEngine.URL,
			Timeout: time.Second,
		}, noopLogger{}),
		config.Server{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			AllowedOrigins:  []string{testAllowedOrigin},
		},
		noopLogger{},
	)
	wsHandler.now = func() time.Time { return websocketFixedNow }
	wsHandler.pollInterval = 10 * time.Millisecond

	server := httptest.NewServer(http.HandlerFunc(wsHandler.HandleConnection))
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), testWebSocketHeaders())
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(dto.ClientEvent{
		Type:          dto.EventSessionStart,
		EventID:       "evt-session-start",
		CorrelationID: "corr-session-start",
		Timestamp:     websocketFixedNow.Format(time.RFC3339Nano),
		ClientID:      "browser-a",
	}); err != nil {
		t.Fatalf("write session.start event: %v", err)
	}

	started := readEvent[dto.SessionStartedEvent](t, conn)
	if started.Type != dto.EventSessionStarted {
		t.Fatalf("session.started type = %q", started.Type)
	}

	if err := conn.WriteJSON(dto.ClientEvent{
		Type:          dto.EventQuickReplySelected,
		SessionID:     started.SessionID,
		EventID:       "evt-quick-reply",
		CorrelationID: "corr-quick-reply",
		Timestamp:     websocketFixedNow.Format(time.RFC3339Nano),
		QuickReply: &dto.QuickReply{
			ID:     "operator",
			Label:  "Связаться с оператором",
			Action: "request_operator",
		},
	}); err != nil {
		t.Fatalf("write quick_reply.selected event: %v", err)
	}

	queued := readEvent[dto.HandoffEvent](t, conn)
	if queued.Type != dto.EventHandoffQueued {
		t.Fatalf("handoff queued type = %q", queued.Type)
	}

	postJSONRequest(t, decisionEngine.URL+"/api/v1/operator/queue/"+started.SessionID+"/accept", map[string]string{
		"operator_id": "operator-1",
	})

	accepted := readEvent[dto.HandoffEvent](t, conn)
	if accepted.Type != dto.EventHandoffAccepted {
		t.Fatalf("handoff accepted type = %q", accepted.Type)
	}
	if accepted.Handoff.Status != "accepted" {
		t.Fatalf("handoff accepted status = %q", accepted.Handoff.Status)
	}
	if accepted.CorrelationID != accepted.Handoff.HandoffID {
		t.Fatalf("handoff accepted correlation_id = %q, want %q", accepted.CorrelationID, accepted.Handoff.HandoffID)
	}

	postJSONRequest(t, decisionEngine.URL+"/api/v1/operator/sessions/"+started.SessionID+"/messages", map[string]string{
		"operator_id": "operator-1",
		"text":        "Здравствуйте, оператор подключился.",
	})

	operatorMessage := readEvent[dto.MessageOperatorEvent](t, conn)
	if operatorMessage.Type != dto.EventMessageOperator {
		t.Fatalf("message.operator type = %q", operatorMessage.Type)
	}
	if operatorMessage.SessionID != started.SessionID {
		t.Fatalf("message.operator session_id = %q, want %q", operatorMessage.SessionID, started.SessionID)
	}
	if operatorMessage.MessageID != "33333333-3333-3333-3333-333333333333" {
		t.Fatalf("message.operator message_id = %q", operatorMessage.MessageID)
	}
	if operatorMessage.CorrelationID != operatorMessage.MessageID {
		t.Fatalf("message.operator correlation_id = %q, want message_id", operatorMessage.CorrelationID)
	}
	if operatorMessage.Text != "Здравствуйте, оператор подключился." {
		t.Fatalf("message.operator text = %q", operatorMessage.Text)
	}
}

func TestHandleConnectionAcceptsAllowedOrigin(t *testing.T) {
	t.Parallel()

	wsHandler := NewHandler(
		client.NewClient(config.DecisionEngine{
			URL:     "http://127.0.0.1:1",
			Timeout: 100 * time.Millisecond,
		}, noopLogger{}),
		config.Server{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			AllowedOrigins:  []string{testAllowedOrigin},
		},
		noopLogger{},
	)

	server := httptest.NewServer(http.HandlerFunc(wsHandler.HandleConnection))
	defer server.Close()

	headers := http.Header{}
	headers.Set("Origin", testAllowedOrigin)

	conn, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), headers)
	if err != nil {
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}
		t.Fatalf("dial websocket with allowed origin: err=%v status=%d", err, statusCode)
	}
	defer conn.Close()
}

func TestHandleConnectionRejectsDisallowedOriginWithSafeResponse(t *testing.T) {
	t.Parallel()

	log := &capturingLogger{}
	wsHandler := NewHandler(
		client.NewClient(config.DecisionEngine{
			URL:     "http://127.0.0.1:1",
			Timeout: 100 * time.Millisecond,
		}, noopLogger{}),
		config.Server{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			AllowedOrigins:  []string{testAllowedOrigin},
		},
		log,
	)

	server := httptest.NewServer(http.HandlerFunc(wsHandler.HandleConnection))
	defer server.Close()

	headers := http.Header{}
	headers.Set("Origin", "https://evil.example.test")

	conn, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), headers)
	if err == nil {
		conn.Close()
		t.Fatal("expected disallowed origin dial to fail")
	}
	if resp == nil {
		t.Fatalf("expected HTTP response for disallowed origin, got err=%v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("disallowed origin status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		t.Fatalf("read disallowed origin body: %v", readErr)
	}
	if got := string(body); got != "Forbidden\n" {
		t.Fatalf("disallowed origin body = %q, want %q", got, "Forbidden\n")
	}
	if strings.Contains(string(body), "origin") || strings.Contains(string(body), "websocket") {
		t.Fatalf("disallowed origin body exposes internal details: %q", string(body))
	}

	entries := log.entries()
	if len(entries) != 1 {
		t.Fatalf("captured log entries = %d, want 1", len(entries))
	}
	entry := entries[0]
	if entry.level != "warn" {
		t.Fatalf("log level = %q, want warn", entry.level)
	}
	if entry.message != "websocket origin rejected" {
		t.Fatalf("log message = %q", entry.message)
	}
	if entry.fields["origin"] != "https://evil.example.test" {
		t.Fatalf("logged origin = %v", entry.fields["origin"])
	}
	if entry.fields["host"] == "" {
		t.Fatalf("logged host is empty: %#v", entry.fields)
	}
	if entry.fields["remote_addr"] == "" {
		t.Fatalf("logged remote_addr is empty: %#v", entry.fields)
	}
}

func readEvent[T any](t *testing.T, conn *websocket.Conn) T {
	t.Helper()

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	var event T
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read websocket event: %v", err)
	}
	return event
}

func dialTestWebSocket(t *testing.T, decisionEngineURL string) *websocket.Conn {
	t.Helper()

	wsHandler := NewHandler(
		client.NewClient(config.DecisionEngine{
			URL:     decisionEngineURL,
			Timeout: 2 * time.Second,
		}, noopLogger{}),
		config.Server{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			AllowedOrigins:  []string{testAllowedOrigin},
		},
		noopLogger{},
	)
	wsHandler.now = func() time.Time { return websocketFixedNow }

	server := httptest.NewServer(http.HandlerFunc(wsHandler.HandleConnection))
	t.Cleanup(server.Close)

	conn, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), testWebSocketHeaders())
	if err != nil {
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}
		t.Fatalf("dial websocket: err=%v status=%d", err, statusCode)
	}
	return conn
}

func testWebSocketHeaders() http.Header {
	headers := http.Header{}
	headers.Set("Origin", testAllowedOrigin)
	return headers
}

func webSocketContractPath() string {
	return filepath.Join("..", "..", "..", "contracts", "websocket.json")
}

func loadWebSocketContract(t *testing.T) map[string]any {
	t.Helper()

	data, err := os.ReadFile(webSocketContractPath())
	if err != nil {
		t.Fatalf("read websocket contract: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal websocket contract: %v", err)
	}
	return doc
}

func postJSONRequest(t *testing.T, requestURL string, payload any) {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request payload: %v", err)
	}

	resp, err := http.Post(requestURL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post %s: %v", requestURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		t.Fatalf("post %s status = %d", requestURL, resp.StatusCode)
	}
}

type noopLogger struct{}

func (noopLogger) Debug(string, ...logger.Field) {}
func (noopLogger) Info(string, ...logger.Field)  {}
func (noopLogger) Warn(string, ...logger.Field)  {}
func (noopLogger) Error(string, ...logger.Field) {}

type logEntry struct {
	level   string
	message string
	fields  map[string]any
}

type capturingLogger struct {
	mu          sync.Mutex
	entriesList []logEntry
}

func (l *capturingLogger) Debug(msg string, fields ...logger.Field) {
	l.append("debug", msg, fields...)
}

func (l *capturingLogger) Info(msg string, fields ...logger.Field) {
	l.append("info", msg, fields...)
}

func (l *capturingLogger) Warn(msg string, fields ...logger.Field) {
	l.append("warn", msg, fields...)
}

func (l *capturingLogger) Error(msg string, fields ...logger.Field) {
	l.append("error", msg, fields...)
}

func (l *capturingLogger) entries() []logEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	cloned := make([]logEntry, len(l.entriesList))
	copy(cloned, l.entriesList)
	return cloned
}

func (l *capturingLogger) append(level string, msg string, fields ...logger.Field) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entryFields := make(map[string]any, len(fields))
	for _, field := range fields {
		entryFields[field.Key] = field.Value
	}
	l.entriesList = append(l.entriesList, logEntry{
		level:   level,
		message: msg,
		fields:  entryFields,
	})
}
