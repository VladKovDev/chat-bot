package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/VladKovDev/chat-bot/internal/apperror"
	"github.com/VladKovDev/chat-bot/internal/contracts"
	"github.com/VladKovDev/chat-bot/internal/domain/message"
	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var fixedNow = time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

func TestV1ContractDocumentListsRequiredRoutes(t *testing.T) {
	t.Parallel()

	doc := loadJSONFile(t, contractPath("http-v1.json"))

	routes, ok := doc["routes"].(map[string]any)
	if !ok {
		t.Fatalf("routes is not an object: %#v", doc["routes"])
	}

	expected := map[string]string{
		"health":           "/api/v1/health",
		"ready":            "/api/v1/ready",
		"sessions":         "/api/v1/sessions",
		"messages":         "/api/v1/messages",
		"session_messages": "/api/v1/sessions/{session_id}/messages",
		"domain_schema":    "/api/v1/domain/schema",
		"operator_request": "/api/v1/operator/queue/{session_id}/request",
		"operator_queue":   "/api/v1/operator/queue",
		"operator_accept":  "/api/v1/operator/queue/{handoff_id}/accept",
		"operator_message": "/api/v1/operator/sessions/{session_id}/messages",
		"operator_close":   "/api/v1/operator/queue/{handoff_id}/close",
	}

	for key, wantPath := range expected {
		rawRoute, ok := routes[key].(map[string]any)
		if !ok {
			t.Fatalf("route %q missing from contract document", key)
		}
		if gotPath, _ := rawRoute["path"].(string); gotPath != wantPath {
			t.Fatalf("route %q path = %q, want %q", key, gotPath, wantPath)
		}
	}

	docBytes, err := os.ReadFile(contractPath("http-v1.json"))
	if err != nil {
		t.Fatalf("read contract file: %v", err)
	}
	text := string(docBytes)
	for _, forbidden := range []string{"/config_llm", "\"/decide\""} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("contract document still references legacy endpoint %q", forbidden)
		}
	}
}

func TestStartSessionUsesVersionedShapeAndResumedFlag(t *testing.T) {
	t.Parallel()

	handler, store, _, _ := newTestHandler()

	first := postJSON[StartSessionResponse](t, handler.StartSession, "/api/v1/sessions", StartSessionRequest{
		Channel:  session.ChannelWebsite,
		ClientID: "browser-a",
	})
	if first.Resumed {
		t.Fatalf("first session should not be resumed: %+v", first)
	}
	if first.Mode != string(session.ModeStandard) {
		t.Fatalf("first mode = %q, want %q", first.Mode, session.ModeStandard)
	}
	if first.UserID == "" || first.SessionID == "" {
		t.Fatalf("first session IDs are empty: %+v", first)
	}

	second := postJSON[StartSessionResponse](t, handler.StartSession, "/api/v1/sessions", StartSessionRequest{
		Channel:  session.ChannelWebsite,
		ClientID: "browser-a",
	})
	if !second.Resumed {
		t.Fatalf("second session should be resumed: %+v", second)
	}
	if second.SessionID != first.SessionID {
		t.Fatalf("resumed session mismatch: first=%s second=%s", first.SessionID, second.SessionID)
	}

	if len(store.sessions) != 1 {
		t.Fatalf("session store size = %d, want 1", len(store.sessions))
	}
}

func TestMessageReturnsVersionedPayloadAndStableErrorEnvelope(t *testing.T) {
	t.Parallel()

	handler, store, _, worker := newTestHandler()
	sessionID := seedSession(t, store, session.Identity{Channel: session.ChannelWebsite, ClientID: "browser-a"})
	worker.response = response.Response{
		SessionID:     sessionID,
		UserMessageID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		BotMessageID:  uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Mode:          session.ModeStandard,
		ActiveTopic:   string(state.StatePayment),
		Text:          "Оплата найдена.",
		Options:       []string{"Связаться с оператором"},
	}

	req := MessageRequest{
		SessionID: sessionID.String(),
		Type:      httpMessageTypeUser,
		Text:      "Проверьте оплату",
		Channel:   session.ChannelWebsite,
		ClientID:  "browser-a",
		EventID:   "33333333-3333-3333-3333-333333333333",
	}
	resp := postJSONWithRequestID[MessageResponse](t, handler.Message, "/api/v1/messages", req, "req-message-1")

	if resp.SessionID != sessionID.String() {
		t.Fatalf("session_id = %q, want %q", resp.SessionID, sessionID)
	}
	if resp.CorrelationID != "req-message-1" {
		t.Fatalf("correlation_id = %q, want req-message-1", resp.CorrelationID)
	}
	if resp.Mode != string(session.ModeStandard) {
		t.Fatalf("mode = %q, want %q", resp.Mode, session.ModeStandard)
	}
	if resp.ActiveTopic == nil || *resp.ActiveTopic != string(state.StatePayment) {
		t.Fatalf("active_topic = %#v, want %q", resp.ActiveTopic, state.StatePayment)
	}
	if len(resp.QuickReplies) != 1 {
		t.Fatalf("quick replies = %#v, want 1 item", resp.QuickReplies)
	}
	quickReply := resp.QuickReplies[0]
	if quickReply.Action != quickReplyActionSend {
		t.Fatalf("quick reply action = %q, want %q", quickReply.Action, quickReplyActionSend)
	}
	if got := quickReply.Payload["text"]; got != "Связаться с оператором" {
		t.Fatalf("quick reply payload text = %#v", got)
	}
	if resp.Handoff != nil {
		t.Fatalf("handoff = %#v, want nil", resp.Handoff)
	}
	if _, err := time.Parse(time.RFC3339Nano, resp.Timestamp); err != nil {
		t.Fatalf("timestamp is not RFC3339: %v", err)
	}

	rawErr := errors.New("pq: syntax error near SELECT * FROM messages; upstream body contains prompt=secret")
	worker.err = apperror.Wrap(apperror.CodeDatabaseUnavailable, "save_message", rawErr)
	rec := httptest.NewRecorder()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, "/api/v1/messages", bytes.NewReader(body))
	httpReq.Header.Set("X-Request-ID", "req-public-1")

	handler.Message(rec, httpReq)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
	responseText := rec.Body.String()
	for _, forbidden := range []string{"SELECT", "upstream body", "prompt=secret", "pq:"} {
		if strings.Contains(responseText, forbidden) {
			t.Fatalf("public error leaked %q: %s", forbidden, responseText)
		}
	}

	var envelope apperror.Envelope
	if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if envelope.Error.Code != apperror.CodeDatabaseUnavailable {
		t.Fatalf("error code = %q, want %q", envelope.Error.Code, apperror.CodeDatabaseUnavailable)
	}
	if envelope.Error.RequestID != "req-public-1" {
		t.Fatalf("request_id = %q, want req-public-1", envelope.Error.RequestID)
	}
}

func TestHealthReadyAndDomainSchemaUseV1Shapes(t *testing.T) {
	t.Parallel()

	handler, _, _, _ := newTestHandler()
	handler.ready = func(context.Context) ReadyResponse {
		return ReadyResponse{
			Ready:     true,
			Timestamp: fixedNow.Format(time.RFC3339Nano),
			Checks: map[string]ReadinessItem{
				"database":   {Ready: true, Message: "ok"},
				"migrations": {Ready: true, Message: "ok"},
				"nlp":        {Ready: true, Message: "ok"},
				"pgvector":   {Ready: true, Message: "ok"},
				"seed_data":  {Ready: true, Message: "ok"},
			},
		}
	}

	health := runNoBody[HealthResponse](t, handler.Health, "/api/v1/health")
	if health.Status != "ok" {
		t.Fatalf("health status = %q, want ok", health.Status)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ready", nil)
	handler.Ready(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ready status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var ready ReadyResponse
	if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&ready); err != nil {
		t.Fatalf("decode ready response: %v", err)
	}
	if !ready.Ready {
		t.Fatalf("ready response should be ready: %+v", ready)
	}
	if len(ready.Checks) != 5 {
		t.Fatalf("ready checks = %#v, want 5 keys", ready.Checks)
	}

	schema := runNoBody[DomainSchemaResponse](t, handler.DomainSchema, "/api/v1/domain/schema")
	if slices.Contains(schema.Actions, "config_llm") {
		t.Fatalf("domain schema leaked config_llm naming: %#v", schema.Actions)
	}
	if !slices.Equal(schema.WebSocketEvents.Client, []string{
		"session.start",
		"message.user",
		"quick_reply.selected",
		"operator.close",
	}) {
		t.Fatalf("client websocket events = %#v", schema.WebSocketEvents.Client)
	}
	if !slices.Contains(schema.WebSocketEvents.Server, "message.operator") {
		t.Fatalf("server websocket events = %#v", schema.WebSocketEvents.Server)
	}
}

func TestOperatorQueueLifecycleAndHistoryEndpoints(t *testing.T) {
	t.Parallel()

	handler, store, messageStore, _ := newTestHandler()
	sessionID := seedSession(t, store, session.Identity{Channel: session.ChannelWebsite, ClientID: "browser-a"})
	store.mustSetSession(session.Session{
		ID:             sessionID,
		UserID:         store.sessions[sessionID].UserID,
		Channel:        session.ChannelWebsite,
		ClientID:       "browser-a",
		State:          state.StatePayment,
		Mode:           session.ModeStandard,
		Status:         session.StatusActive,
		OperatorStatus: session.OperatorStatusNone,
		ActiveTopic:    string(state.StatePayment),
		LastIntent:     "payment_not_activated",
		Metadata:       map[string]interface{}{},
		CreatedAt:      fixedNow.Add(-5 * time.Minute),
		UpdatedAt:      fixedNow.Add(-5 * time.Minute),
	})

	seededMessage := messageStore.mustCreate(message.Message{
		SessionID:  sessionID,
		SenderType: message.SenderTypeUser,
		Text:       "Деньги списались, услуга не активировалась",
		CreatedAt:  fixedNow.Add(-4 * time.Minute),
	})

	requestResp := postJSON[OperatorQueueActionResponse](t, withURLParam("session_id", sessionID.String(), handler.RequestOperator), "/api/v1/operator/queue/"+sessionID.String()+"/request", RequestOperatorBody{
		Reason: "manual_request",
	})
	if requestResp.Handoff.Status != operatorQueueStatusWaiting {
		t.Fatalf("request handoff status = %q, want %q", requestResp.Handoff.Status, operatorQueueStatusWaiting)
	}

	queueRec := httptest.NewRecorder()
	queueReq := httptest.NewRequest(http.MethodGet, "/api/v1/operator/queue?status=waiting", nil)
	handler.OperatorQueue(queueRec, queueReq)
	if queueRec.Code != http.StatusOK {
		t.Fatalf("operator queue status = %d, body=%s", queueRec.Code, queueRec.Body.String())
	}
	var queue OperatorQueueResponse
	if err := json.NewDecoder(bytes.NewReader(queueRec.Body.Bytes())).Decode(&queue); err != nil {
		t.Fatalf("decode operator queue: %v", err)
	}
	if len(queue.Items) != 1 {
		t.Fatalf("operator queue items = %#v, want 1 item", queue.Items)
	}
	if queue.Items[0].Preview != seededMessage.Text {
		t.Fatalf("queue preview = %q, want %q", queue.Items[0].Preview, seededMessage.Text)
	}

	acceptResp := postJSON[OperatorQueueActionResponse](t, withURLParam("handoff_id", sessionID.String(), handler.AcceptOperatorQueue), "/api/v1/operator/queue/"+sessionID.String()+"/accept", OperatorQueueActionRequest{
		OperatorID: "operator-1",
	})
	if acceptResp.Handoff.Status != operatorQueueStatusAccepted {
		t.Fatalf("accept handoff status = %q, want %q", acceptResp.Handoff.Status, operatorQueueStatusAccepted)
	}
	if acceptResp.Handoff.OperatorID == nil || *acceptResp.Handoff.OperatorID != "operator-1" {
		t.Fatalf("accept operator_id = %#v", acceptResp.Handoff.OperatorID)
	}

	operatorMessage := postJSON[OperatorMessageResponse](t, withURLParam("session_id", sessionID.String(), handler.OperatorMessage), "/api/v1/operator/sessions/"+sessionID.String()+"/messages", OperatorMessageRequest{
		OperatorID: "operator-1",
		Text:       "Здравствуйте, я подключился.",
	})
	if operatorMessage.OperatorID != "operator-1" {
		t.Fatalf("operator message operator_id = %q", operatorMessage.OperatorID)
	}
	if operatorMessage.SessionID != sessionID.String() {
		t.Fatalf("operator message session_id = %q, want %q", operatorMessage.SessionID, sessionID)
	}

	historyRec := httptest.NewRecorder()
	historyReq := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+sessionID.String()+"/messages", nil)
	withURLParam("session_id", sessionID.String(), handler.SessionMessages)(historyRec, historyReq)
	if historyRec.Code != http.StatusOK {
		t.Fatalf("history status = %d, body=%s", historyRec.Code, historyRec.Body.String())
	}
	var history SessionMessagesResponse
	if err := json.NewDecoder(bytes.NewReader(historyRec.Body.Bytes())).Decode(&history); err != nil {
		t.Fatalf("decode history: %v", err)
	}
	if len(history.Items) != 2 {
		t.Fatalf("history items = %#v, want 2 items", history.Items)
	}
	if history.Items[1].SenderType != string(message.SenderTypeOperator) {
		t.Fatalf("history sender_type = %q, want %q", history.Items[1].SenderType, message.SenderTypeOperator)
	}

	closeResp := postJSON[OperatorQueueActionResponse](t, withURLParam("handoff_id", sessionID.String(), handler.CloseOperatorQueue), "/api/v1/operator/queue/"+sessionID.String()+"/close", OperatorQueueActionRequest{
		OperatorID: "operator-1",
	})
	if closeResp.Handoff.Status != operatorQueueStatusClosed {
		t.Fatalf("close handoff status = %q, want %q", closeResp.Handoff.Status, operatorQueueStatusClosed)
	}
}

func newTestHandler() (*Handler, *fakeSessionStore, *fakeMessageStore, *fakeWorker) {
	sessionStore := newFakeSessionStore()
	messageStore := newFakeMessageStore()
	worker := &fakeWorker{}
	handler := NewHandler(worker, sessionStore, sessionStore, messageStore, logger.Noop())
	handler.now = func() time.Time { return fixedNow }
	return handler, sessionStore, messageStore, worker
}

type fakeWorker struct {
	response response.Response
	err      error
}

func (f *fakeWorker) HandleMessage(_ context.Context, _ contracts.IncomingMessage) (response.Response, error) {
	if f.err != nil {
		return response.Response{}, f.err
	}
	return f.response, nil
}

type fakeSessionStore struct {
	sessions   map[uuid.UUID]session.Session
	byIdentity map[string]uuid.UUID
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{
		sessions:   make(map[uuid.UUID]session.Session),
		byIdentity: make(map[string]uuid.UUID),
	}
}

func (f *fakeSessionStore) StartSession(_ context.Context, identity session.Identity) (session.StartResult, error) {
	key := identityKey(identity)
	if sessionID, ok := f.byIdentity[key]; ok {
		return session.StartResult{Session: f.sessions[sessionID], Resumed: true}, nil
	}

	sess := session.Session{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		Channel:        identity.Channel,
		ExternalUserID: identity.ExternalUserID,
		ClientID:       identity.ClientID,
		State:          state.StateNew,
		Mode:           session.ModeStandard,
		Status:         session.StatusActive,
		OperatorStatus: session.OperatorStatusNone,
		Metadata:       map[string]interface{}{},
		CreatedAt:      fixedNow,
		UpdatedAt:      fixedNow,
	}
	f.sessions[sess.ID] = sess
	f.byIdentity[key] = sess.ID
	return session.StartResult{Session: sess}, nil
}

func (f *fakeSessionStore) LoadSessionByID(_ context.Context, sessionID uuid.UUID, _ session.Identity) (*session.Session, error) {
	sess, ok := f.sessions[sessionID]
	if !ok {
		return nil, session.ErrNotFound
	}
	return &sess, nil
}

func (f *fakeSessionStore) ApplyContextDecision(_ context.Context, sess *session.Session, decision session.ContextDecision) (session.Session, error) {
	current := *sess
	if current.Metadata == nil {
		current.Metadata = map[string]interface{}{}
	}
	for key, value := range decision.Metadata {
		current.Metadata[key] = value
	}
	switch decision.Event {
	case session.EventRequestOperator:
		current.Mode = session.ModeWaitingOperator
		current.OperatorStatus = session.OperatorStatusWaiting
	case session.EventOperatorConnected:
		current.Mode = session.ModeOperatorConnected
		current.OperatorStatus = session.OperatorStatusConnected
		current.Metadata["operator_id"] = decision.Metadata["operator_id"]
	case session.EventOperatorClosed:
		current.Mode = session.ModeClosed
		current.Status = session.StatusClosed
		current.OperatorStatus = session.OperatorStatusClosed
	}
	if reason, ok := decision.Metadata["handoff_reason"].(string); ok {
		current.Metadata["handoff_reason"] = reason
	}
	current.UpdatedAt = fixedNow
	f.sessions[current.ID] = current
	*sess = current
	return current, nil
}

func (f *fakeSessionStore) GetByID(_ context.Context, id uuid.UUID) (session.Session, error) {
	sess, ok := f.sessions[id]
	if !ok {
		return session.Session{}, session.ErrNotFound
	}
	return sess, nil
}

func (f *fakeSessionStore) ListByStatus(_ context.Context, status session.Status, _ int32, _ int32) ([]session.Session, error) {
	result := make([]session.Session, 0, len(f.sessions))
	for _, sess := range f.sessions {
		if sess.Status == status {
			result = append(result, sess)
		}
	}
	return result, nil
}

func (f *fakeSessionStore) mustSetSession(sess session.Session) {
	if sess.CreatedAt.IsZero() {
		sess.CreatedAt = fixedNow
	}
	if sess.UpdatedAt.IsZero() {
		sess.UpdatedAt = fixedNow
	}
	f.sessions[sess.ID] = sess
	f.byIdentity[identityKey(session.Identity{
		Channel:        sess.Channel,
		ExternalUserID: sess.ExternalUserID,
		ClientID:       sess.ClientID,
	})] = sess.ID
}

type fakeMessageStore struct {
	items map[uuid.UUID][]message.Message
}

func newFakeMessageStore() *fakeMessageStore {
	return &fakeMessageStore{
		items: make(map[uuid.UUID][]message.Message),
	}
}

func (f *fakeMessageStore) Create(_ context.Context, msg message.Message) (message.Message, error) {
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	f.items[msg.SessionID] = append(f.items[msg.SessionID], msg)
	return msg, nil
}

func (f *fakeMessageStore) GetBySessionID(_ context.Context, sessionID uuid.UUID, _ int32, _ int32) ([]message.Message, error) {
	return append([]message.Message(nil), f.items[sessionID]...), nil
}

func (f *fakeMessageStore) GetLastMessagesBySessionID(_ context.Context, sessionID uuid.UUID, limit int32) ([]message.Message, error) {
	items := f.items[sessionID]
	if int(limit) >= len(items) {
		return append([]message.Message(nil), items...), nil
	}
	return append([]message.Message(nil), items[len(items)-int(limit):]...), nil
}

func (f *fakeMessageStore) mustCreate(msg message.Message) message.Message {
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	f.items[msg.SessionID] = append(f.items[msg.SessionID], msg)
	return msg
}

func postJSON[T any](t *testing.T, handler http.HandlerFunc, path string, body any) T {
	t.Helper()
	return postJSONWithRequestID[T](t, handler, path, body, "req-test")
}

func postJSONWithRequestID[T any](t *testing.T, handler http.HandlerFunc, path string, body any, requestID string) T {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", requestID)

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp T
	if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func runNoBody[T any](t *testing.T, handler http.HandlerFunc, path string) T {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp T
	if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func withURLParam(key, value string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		routeCtx := chi.NewRouteContext()
		routeCtx.URLParams.Add(key, value)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))
		handler(w, r)
	}
}

func seedSession(t *testing.T, store *fakeSessionStore, identity session.Identity) uuid.UUID {
	t.Helper()

	result, err := store.StartSession(context.Background(), identity)
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	return result.Session.ID
}

func identityKey(identity session.Identity) string {
	return identity.Channel + "|" + identity.ExternalUserID + "|" + identity.ClientID
}

func contractPath(name string) string {
	return filepath.Join("..", "..", "..", "..", "contracts", name)
}

func loadJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return doc
}
