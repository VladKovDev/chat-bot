package handler

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appactions "github.com/VladKovDev/chat-bot/internal/app/actions"
	appdecision "github.com/VladKovDev/chat-bot/internal/app/decision"
	apppresenter "github.com/VladKovDev/chat-bot/internal/app/presenter"
	appprocessor "github.com/VladKovDev/chat-bot/internal/app/processor"
	appworker "github.com/VladKovDev/chat-bot/internal/app/worker"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/message"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/pkg/logger"
	"github.com/google/uuid"
)

type fakeSemanticMatcher struct {
	result appdecision.MatchResult
	calls  int
}

func (m *fakeSemanticMatcher) Match(
	_ context.Context,
	_ string,
	_ []apppresenter.IntentDefinition,
) (appdecision.MatchResult, error) {
	m.calls++
	return m.result, nil
}

func TestMessageUsesDeterministicDecisionServiceWithoutLLM(t *testing.T) {
	t.Parallel()

	sessionStore := newFakeSessionStore()
	messageStore := newFakeMessageStore()
	sessionService := session.NewService(sessionStore)

	configPath := filepath.Join("..", "..", "..", "..", "configs")
	presenter, err := apppresenter.NewPresenter(configPath)
	if err != nil {
		t.Fatalf("new presenter: %v", err)
	}
	intentCatalog, err := apppresenter.LoadIntentCatalog(configPath)
	if err != nil {
		t.Fatalf("load intent catalog: %v", err)
	}

	matcher := &fakeSemanticMatcher{
		result: appdecision.MatchResult{
			IntentKey:  "ask_payment_status",
			Confidence: 0.92,
			Candidates: []appdecision.Candidate{
				{IntentKey: "ask_payment_status", Confidence: 0.92},
			},
		},
	}
	decisionService, err := appdecision.NewService(intentCatalog, matcher, logger.Noop())
	if err != nil {
		t.Fatalf("new decision service: %v", err)
	}

	processor := appprocessor.NewProcessor(logger.Noop())
	processor.Register("find_payment", appactions.NewFindPayment(logger.Noop()))

	worker := appworker.NewMessageWorker(
		sessionService,
		decisionService,
		processor,
		presenter,
		newIntegrationMessagePersistence(messageStore, sessionStore),
		logger.Noop(),
	)
	handler := NewHandler(worker, sessionService, sessionStore, messageStore, logger.Noop())
	handler.now = func() time.Time { return fixedNow }

	resp := postJSON[MessageResponse](t, handler.Message, "/api/v1/messages", MessageRequest{
		Type:     httpMessageTypeUser,
		Text:     "Проверь оплату PAY-123456",
		Channel:  session.ChannelWebsite,
		ClientID: "browser-a",
	})

	if matcher.calls != 1 {
		t.Fatalf("matcher calls = %d, want 1", matcher.calls)
	}
	if !strings.Contains(resp.Text, "Платёж найден") {
		t.Fatalf("response text = %q, want payment result", resp.Text)
	}

	sessionID := uuid.MustParse(resp.SessionID)
	items := messageStore.items[sessionID]
	if len(items) != 2 {
		t.Fatalf("message count = %d, want 2", len(items))
	}
	if items[0].SenderType != message.SenderTypeUser {
		t.Fatalf("first sender = %q, want %q", items[0].SenderType, message.SenderTypeUser)
	}
	if items[1].SenderType != message.SenderTypeBot {
		t.Fatalf("second sender = %q, want %q", items[1].SenderType, message.SenderTypeBot)
	}

	sess := sessionStore.sessions[sessionID]
	if sess.State != state.StatePayment {
		t.Fatalf("session state = %q, want %q", sess.State, state.StatePayment)
	}
	if sess.ActiveTopic != "payment" {
		t.Fatalf("active topic = %q, want payment", sess.ActiveTopic)
	}
	if sess.LastIntent != "ask_payment_status" {
		t.Fatalf("last intent = %q, want ask_payment_status", sess.LastIntent)
	}
	if sess.Mode != session.ModeStandard {
		t.Fatalf("mode = %q, want %q", sess.Mode, session.ModeStandard)
	}
}

func (f *fakeSessionStore) Create(_ context.Context, sess session.Session) (session.Session, error) {
	if sess.ID == uuid.Nil {
		sess.ID = uuid.New()
	}
	if sess.UserID == uuid.Nil {
		sess.UserID = uuid.New()
	}
	if sess.Metadata == nil {
		sess.Metadata = map[string]interface{}{}
	}
	if sess.Status == "" {
		sess.Status = session.StatusActive
	}
	if sess.Mode == "" {
		sess.Mode = session.ModeStandard
	}
	if sess.OperatorStatus == "" {
		sess.OperatorStatus = session.OperatorStatusNone
	}
	if sess.CreatedAt.IsZero() {
		sess.CreatedAt = fixedNow
	}
	if sess.UpdatedAt.IsZero() {
		sess.UpdatedAt = fixedNow
	}
	f.mustSetSession(sess)
	return sess, nil
}

func (f *fakeSessionStore) GetActiveByIdentity(_ context.Context, identity session.Identity) (session.Session, error) {
	sessionID, ok := f.byIdentity[identityKey(identity)]
	if !ok {
		return session.Session{}, session.ErrNotFound
	}
	sess := f.sessions[sessionID]
	if sess.Status != session.StatusActive {
		return session.Session{}, session.ErrNotFound
	}
	return sess, nil
}

func (f *fakeSessionStore) GetByUserID(_ context.Context, userID uuid.UUID, _ int32, _ int32) ([]session.Session, error) {
	result := make([]session.Session, 0)
	for _, sess := range f.sessions {
		if sess.UserID == userID {
			result = append(result, sess)
		}
	}
	return result, nil
}

func (f *fakeSessionStore) Update(_ context.Context, sess session.Session) (session.Session, error) {
	sess.UpdatedAt = fixedNow
	f.mustSetSession(sess)
	return sess, nil
}

func (f *fakeSessionStore) UpdateContext(_ context.Context, sess session.Session, _ *session.ModeTransition) (session.Session, error) {
	sess.UpdatedAt = fixedNow
	f.mustSetSession(sess)
	return sess, nil
}

func (f *fakeSessionStore) UpdateState(_ context.Context, id uuid.UUID, st state.State) (session.Session, error) {
	sess, ok := f.sessions[id]
	if !ok {
		return session.Session{}, session.ErrNotFound
	}
	sess.State = st
	sess.UpdatedAt = fixedNow
	f.mustSetSession(sess)
	return sess, nil
}

func (f *fakeSessionStore) UpdateStateWithVersion(ctx context.Context, id uuid.UUID, st state.State) (session.Session, error) {
	sess, err := f.UpdateState(ctx, id, st)
	if err != nil {
		return session.Session{}, err
	}
	sess.Version++
	f.mustSetSession(sess)
	return sess, nil
}

func (f *fakeSessionStore) UpdateStatus(_ context.Context, id uuid.UUID, status session.Status) (session.Session, error) {
	sess, ok := f.sessions[id]
	if !ok {
		return session.Session{}, session.ErrNotFound
	}
	sess.Status = status
	sess.UpdatedAt = fixedNow
	f.mustSetSession(sess)
	return sess, nil
}

func (f *fakeSessionStore) List(_ context.Context, _ int32, _ int32) ([]session.Session, error) {
	result := make([]session.Session, 0, len(f.sessions))
	for _, sess := range f.sessions {
		result = append(result, sess)
	}
	return result, nil
}

func (f *fakeSessionStore) ListByState(_ context.Context, st state.State, _ int32, _ int32) ([]session.Session, error) {
	result := make([]session.Session, 0)
	for _, sess := range f.sessions {
		if sess.State == st {
			result = append(result, sess)
		}
	}
	return result, nil
}

func (f *fakeSessionStore) Delete(_ context.Context, id uuid.UUID) error {
	delete(f.sessions, id)
	return nil
}

func (f *fakeSessionStore) Count(_ context.Context) (int64, error) {
	return int64(len(f.sessions)), nil
}

func (f *fakeMessageStore) CountBySessionID(_ context.Context, sessionID uuid.UUID) (int64, error) {
	return int64(len(f.items[sessionID])), nil
}

type integrationMessagePersistence struct {
	messages *fakeMessageStore
	sessions *fakeSessionStore
}

func newIntegrationMessagePersistence(
	messages *fakeMessageStore,
	sessions *fakeSessionStore,
) *integrationMessagePersistence {
	return &integrationMessagePersistence{
		messages: messages,
		sessions: sessions,
	}
}

func (p *integrationMessagePersistence) WithinMessageTransaction(
	ctx context.Context,
	fn func(context.Context, appworker.MessageTransaction) error,
) error {
	return fn(ctx, &integrationMessageTx{messages: p.messages, sessions: p.sessions})
}

type integrationMessageTx struct {
	messages *fakeMessageStore
	sessions *fakeSessionStore
}

func (tx *integrationMessageTx) CreateMessage(ctx context.Context, msg message.Message) (message.Message, error) {
	return tx.messages.Create(ctx, msg)
}

func (tx *integrationMessageTx) GetLastMessagesBySessionID(
	ctx context.Context,
	sessionID uuid.UUID,
	limit int32,
) ([]message.Message, error) {
	return tx.messages.GetLastMessagesBySessionID(ctx, sessionID, limit)
}

func (tx *integrationMessageTx) LogDecision(_ context.Context, _ appworker.DecisionLog) error {
	return nil
}

func (tx *integrationMessageTx) LogAction(_ context.Context, _ action.Log) error {
	return nil
}

func (tx *integrationMessageTx) ApplyContextDecision(
	_ context.Context,
	sess *session.Session,
	decision session.ContextDecision,
) (session.Session, error) {
	next, _, err := session.PrepareContextUpdate(sess, decision)
	if err != nil {
		return session.Session{}, err
	}
	tx.sessions.mustSetSession(next)
	*sess = next
	return next, nil
}

var _ appworker.MessagePersistence = (*integrationMessagePersistence)(nil)

var _ appworker.MessageTransaction = (*integrationMessageTx)(nil)
