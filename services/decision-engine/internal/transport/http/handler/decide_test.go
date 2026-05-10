package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/VladKovDev/chat-bot/internal/apperror"
	"github.com/VladKovDev/chat-bot/internal/contracts"
	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/pkg/logger"
	"github.com/google/uuid"
)

func TestStartSessionReturnsResumedFlag(t *testing.T) {
	t.Parallel()

	worker := newFakeWorker()
	handler := NewHandler(worker, logger.Noop())

	first := postSession(t, handler, StartSessionRequest{
		Channel:  session.ChannelWebsite,
		ClientID: "browser-a",
	})
	if !first.Success || first.Resumed {
		t.Fatalf("first session response = %+v", first)
	}

	second := postSession(t, handler, StartSessionRequest{
		Channel:  session.ChannelWebsite,
		ClientID: "browser-a",
	})
	if !second.Success || !second.Resumed {
		t.Fatalf("second session response = %+v", second)
	}
	if first.SessionID != second.SessionID {
		t.Fatalf("resumed session_id mismatch: first=%s second=%s", first.SessionID, second.SessionID)
	}
}

func TestDecideRequiresIdentityOrExplicitDevCLIChannel(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newFakeWorker(), logger.Noop())

	for _, tc := range []struct {
		name string
		body DecideRequest
	}{
		{name: "empty identity", body: DecideRequest{Text: "hello"}},
		{name: "chat id without dev cli channel", body: DecideRequest{Text: "hello", ChatID: 99}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			body, err := json.Marshal(tc.body)
			if err != nil {
				t.Fatalf("marshal request: %v", err)
			}
			req := httptest.NewRequest(http.MethodPost, "/decide", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			handler.Decide(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
		})
	}
}

func TestDecideReturnsPublicErrorShapeWithoutInternalDetails(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("pq: syntax error near SELECT * FROM messages; upstream body contains prompt=secret")
	worker := newFakeWorker()
	worker.handleErr = apperror.Wrap(apperror.CodeDatabaseUnavailable, "save_message", rawErr)
	handler := NewHandler(worker, logger.Noop())

	body, err := json.Marshal(DecideRequest{
		Text:      "секретный пользовательский текст",
		SessionID: uuid.NewString(),
		Channel:   session.ChannelWebsite,
		ClientID:  "browser-a",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/decide", bytes.NewReader(body))
	req.Header.Set("X-Request-ID", "req-public-1")
	rec := httptest.NewRecorder()

	handler.Decide(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
	bodyText := rec.Body.String()
	for _, forbidden := range []string{"SELECT", "upstream body", "prompt=secret", "pq:", "секретный пользовательский текст"} {
		if strings.Contains(bodyText, forbidden) {
			t.Fatalf("public response leaked %q: %s", forbidden, bodyText)
		}
	}

	var resp DecideResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error == nil {
		t.Fatalf("error is nil in response: %+v", resp)
	}
	if resp.Error.Code != apperror.CodeDatabaseUnavailable {
		t.Fatalf("code = %q, want %q", resp.Error.Code, apperror.CodeDatabaseUnavailable)
	}
	if resp.Error.RequestID != "req-public-1" {
		t.Fatalf("request_id = %q, want req-public-1", resp.Error.RequestID)
	}
	if resp.Error.Message == "" {
		t.Fatalf("safe message is empty")
	}
}

func TestDecideKeepsBrowserSessionsIsolated(t *testing.T) {
	t.Parallel()

	worker := newFakeWorker()
	handler := NewHandler(worker, logger.Noop())

	clientA := postSession(t, handler, StartSessionRequest{Channel: session.ChannelWebsite, ClientID: "browser-a"})
	clientB := postSession(t, handler, StartSessionRequest{Channel: session.ChannelWebsite, ClientID: "browser-b"})
	if clientA.SessionID == clientB.SessionID {
		t.Fatalf("different browser clients got same session_id: %s", clientA.SessionID)
	}

	respA := postDecide(t, handler, DecideRequest{
		Text:      "payment problem",
		SessionID: clientA.SessionID,
		Channel:   session.ChannelWebsite,
		ClientID:  "browser-a",
	})
	respB := postDecide(t, handler, DecideRequest{
		Text:      "workspace problem",
		SessionID: clientB.SessionID,
		Channel:   session.ChannelWebsite,
		ClientID:  "browser-b",
	})

	if respA.SessionID == respB.SessionID {
		t.Fatalf("response session IDs were mixed: %s", respA.SessionID)
	}
	if respA.ActiveTopic != string(state.StatePayment) {
		t.Fatalf("client A active_topic = %q, want payment", respA.ActiveTopic)
	}
	if respB.ActiveTopic != string(state.StateWorkspace) {
		t.Fatalf("client B active_topic = %q, want workspace", respB.ActiveTopic)
	}
	if got := worker.history[clientA.SessionID]; len(got) != 1 || got[0] != "payment problem" {
		t.Fatalf("client A history = %#v", got)
	}
	if got := worker.history[clientB.SessionID]; len(got) != 1 || got[0] != "workspace problem" {
		t.Fatalf("client B history = %#v", got)
	}
}

func postSession(t *testing.T, handler *Handler, body StartSessionRequest) StartSessionResponse {
	t.Helper()

	reqBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal session request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader(reqBody))
	rec := httptest.NewRecorder()

	handler.StartSession(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("session status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var resp StartSessionResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode session response: %v", err)
	}
	return resp
}

func postDecide(t *testing.T, handler *Handler, body DecideRequest) DecideResponse {
	t.Helper()

	reqBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal decide request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/decide", bytes.NewReader(reqBody))
	rec := httptest.NewRecorder()

	handler.Decide(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("decide status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var resp DecideResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode decide response: %v", err)
	}
	return resp
}

type fakeWorker struct {
	sessions  map[string]session.Session
	byClient  map[string]string
	history   map[string][]string
	handleErr error
	startErr  error
}

func newFakeWorker() *fakeWorker {
	return &fakeWorker{
		sessions: make(map[string]session.Session),
		byClient: make(map[string]string),
		history:  make(map[string][]string),
	}
}

func (f *fakeWorker) StartSession(_ context.Context, identity session.Identity) (session.StartResult, error) {
	if f.startErr != nil {
		return session.StartResult{}, f.startErr
	}

	if sessionID, ok := f.byClient[identity.ClientID]; ok {
		return session.StartResult{Session: f.sessions[sessionID], Resumed: true}, nil
	}

	id := uuid.New()
	sess := session.Session{
		ID:       id,
		Channel:  identity.Channel,
		ClientID: identity.ClientID,
		State:    state.StateNew,
		Status:   session.StatusActive,
	}
	sessionID := id.String()
	f.sessions[sessionID] = sess
	f.byClient[identity.ClientID] = sessionID
	return session.StartResult{Session: sess, Resumed: false}, nil
}

func (f *fakeWorker) HandleMessage(_ context.Context, msg contracts.IncomingMessage) (response.Response, error) {
	if f.handleErr != nil {
		return response.Response{}, f.handleErr
	}

	sessionID := msg.SessionID.String()
	f.history[sessionID] = append(f.history[sessionID], msg.Text)

	activeTopic := string(state.StateWorkspace)
	if msg.Text == "payment problem" {
		activeTopic = string(state.StatePayment)
	}

	return response.Response{
		Text:        "ok",
		State:       state.State(activeTopic),
		SessionID:   msg.SessionID,
		Channel:     msg.Channel,
		ClientID:    msg.ClientID,
		ActiveTopic: activeTopic,
	}, nil
}
