package session

import (
	"context"
	"errors"
	"testing"

	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/google/uuid"
)

func TestStartSessionCreatesAndResumesIsolatedBrowserSessions(t *testing.T) {
	t.Parallel()

	repo := newMemoryRepo()
	service := NewService(repo)
	ctx := context.Background()

	clientA := Identity{Channel: ChannelWebsite, ClientID: "browser-a"}
	clientB := Identity{Channel: ChannelWebsite, ClientID: "browser-b"}

	firstA, err := service.StartSession(ctx, clientA)
	if err != nil {
		t.Fatalf("start client A: %v", err)
	}
	if firstA.Resumed {
		t.Fatalf("first client A session unexpectedly resumed")
	}

	firstB, err := service.StartSession(ctx, clientB)
	if err != nil {
		t.Fatalf("start client B: %v", err)
	}
	if firstB.Resumed {
		t.Fatalf("first client B session unexpectedly resumed")
	}
	if firstA.Session.ID == firstB.Session.ID {
		t.Fatalf("different browser clients got the same session_id: %s", firstA.Session.ID)
	}
	if firstA.Session.Mode != ModeStandard {
		t.Fatalf("new session mode = %q, want %q", firstA.Session.Mode, ModeStandard)
	}
	if firstA.Session.UserID == uuid.Nil || firstB.Session.UserID == uuid.Nil {
		t.Fatalf("new sessions should include user IDs: A=%s B=%s", firstA.Session.UserID, firstB.Session.UserID)
	}

	firstA.Session.State = state.StatePayment
	firstA.Session.ActiveTopic = string(state.StatePayment)
	if _, err := service.UpdateSessionState(ctx, &firstA.Session); err != nil {
		t.Fatalf("update client A state: %v", err)
	}

	resumedA, err := service.StartSession(ctx, clientA)
	if err != nil {
		t.Fatalf("resume client A: %v", err)
	}
	if !resumedA.Resumed {
		t.Fatalf("second client A session was not marked resumed")
	}
	if resumedA.Session.ID != firstA.Session.ID {
		t.Fatalf("client A resumed a different session: got %s want %s", resumedA.Session.ID, firstA.Session.ID)
	}
	if resumedA.Session.ActiveTopic != string(state.StatePayment) {
		t.Fatalf("client A active topic was not preserved: got %q", resumedA.Session.ActiveTopic)
	}

	resumedB, err := service.StartSession(ctx, clientB)
	if err != nil {
		t.Fatalf("resume client B: %v", err)
	}
	if resumedB.Session.ActiveTopic != "" || resumedB.Session.State != state.StateNew {
		t.Fatalf("client B leaked client A state/topic: state=%q topic=%q", resumedB.Session.State, resumedB.Session.ActiveTopic)
	}
}

func TestApplyContextDecisionLogsLimitedModeFSM(t *testing.T) {
	t.Parallel()

	repo := newMemoryRepo()
	service := NewService(repo)
	ctx := context.Background()

	started, err := service.StartSession(ctx, Identity{Channel: ChannelWebsite, ClientID: "fsm-client"})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	sess := started.Session

	if _, err := service.ApplyContextDecision(ctx, &sess, ContextDecision{
		Intent: "request_operator",
		Topic:  "complaint",
		Event:  EventRequestOperator,
	}); err != nil {
		t.Fatalf("request operator transition: %v", err)
	}
	if sess.Mode != ModeWaitingOperator || sess.OperatorStatus != OperatorStatusWaiting {
		t.Fatalf("request operator context = mode %q operator_status %q", sess.Mode, sess.OperatorStatus)
	}

	if _, err := service.ApplyContextDecision(ctx, &sess, ContextDecision{
		Event: EventOperatorConnected,
	}); err != nil {
		t.Fatalf("operator connected transition: %v", err)
	}
	if sess.Mode != ModeOperatorConnected || sess.OperatorStatus != OperatorStatusConnected {
		t.Fatalf("operator connected context = mode %q operator_status %q", sess.Mode, sess.OperatorStatus)
	}

	if _, err := service.ApplyContextDecision(ctx, &sess, ContextDecision{
		Event: EventOperatorClosed,
	}); err != nil {
		t.Fatalf("operator closed transition: %v", err)
	}
	if sess.Mode != ModeStandard || sess.OperatorStatus != OperatorStatusNone || sess.Status != StatusActive {
		t.Fatalf("operator closed context = mode %q operator_status %q status %q", sess.Mode, sess.OperatorStatus, sess.Status)
	}

	logs := repo.transitions[sess.ID]
	if len(logs) != 3 {
		t.Fatalf("transition log count = %d, want 3", len(logs))
	}
	assertTransition(t, logs[0], ModeStandard, ModeWaitingOperator, EventRequestOperator)
	assertTransition(t, logs[1], ModeWaitingOperator, ModeOperatorConnected, EventOperatorConnected)
	assertTransition(t, logs[2], ModeOperatorConnected, ModeStandard, EventOperatorClosed)
}

func TestStartSessionRestoresPersistentContextAfterRestart(t *testing.T) {
	t.Parallel()

	repo := newMemoryRepo()
	ctx := context.Background()
	identity := Identity{Channel: ChannelWebsite, ClientID: "restore-client"}

	serviceBeforeRestart := NewService(repo)
	started, err := serviceBeforeRestart.StartSession(ctx, identity)
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	sess := started.Session

	for i := 0; i < 2; i++ {
		if _, err := serviceBeforeRestart.ApplyContextDecision(ctx, &sess, ContextDecision{
			Intent:        "unknown",
			Topic:         "payment",
			LowConfidence: true,
			Event:         EventMessageReceived,
			Metadata: map[string]interface{}{
				"confidence_source": "test",
			},
		}); err != nil {
			t.Fatalf("low confidence context update %d: %v", i+1, err)
		}
	}

	serviceAfterRestart := NewService(repo)
	resumed, err := serviceAfterRestart.StartSession(ctx, identity)
	if err != nil {
		t.Fatalf("resume session: %v", err)
	}
	if !resumed.Resumed || resumed.Session.ID != sess.ID {
		t.Fatalf("session was not restored: resumed=%v got=%s want=%s", resumed.Resumed, resumed.Session.ID, sess.ID)
	}
	if resumed.Session.Mode != ModeStandard ||
		resumed.Session.ActiveTopic != "payment" ||
		resumed.Session.LastIntent != "unknown" ||
		resumed.Session.FallbackCount != 2 {
		t.Fatalf("restored context mismatch: mode=%q topic=%q intent=%q fallback=%d",
			resumed.Session.Mode,
			resumed.Session.ActiveTopic,
			resumed.Session.LastIntent,
			resumed.Session.FallbackCount)
	}
	if resumed.Session.Version <= 1 {
		t.Fatalf("restored version = %d, want incremented version", resumed.Session.Version)
	}
	if resumed.Session.Metadata["confidence_source"] != "test" {
		t.Fatalf("restored metadata confidence_source = %#v", resumed.Session.Metadata["confidence_source"])
	}
}

func TestRepeatedLowConfidenceIncrementsAndHighConfidenceResetsFallbackCount(t *testing.T) {
	t.Parallel()

	service := NewService(newMemoryRepo())
	ctx := context.Background()
	started, err := service.StartSession(ctx, Identity{Channel: ChannelWebsite, ClientID: "fallback-client"})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	sess := started.Session

	for i := 1; i <= 2; i++ {
		if _, err := service.ApplyContextDecision(ctx, &sess, ContextDecision{
			Intent:        "unknown",
			LowConfidence: true,
			Event:         EventMessageReceived,
		}); err != nil {
			t.Fatalf("low confidence update %d: %v", i, err)
		}
		if sess.FallbackCount != i {
			t.Fatalf("fallback_count after low confidence %d = %d", i, sess.FallbackCount)
		}
	}

	if _, err := service.ApplyContextDecision(ctx, &sess, ContextDecision{
		Intent: "payment_status",
		Topic:  "payment",
		Event:  EventMessageReceived,
	}); err != nil {
		t.Fatalf("high confidence update: %v", err)
	}
	if sess.FallbackCount != 0 {
		t.Fatalf("fallback_count after high confidence = %d, want 0", sess.FallbackCount)
	}
}

func TestTopicSwitchResetsStaleFlowButKeepsModeSeparate(t *testing.T) {
	t.Parallel()

	service := NewService(newMemoryRepo())
	ctx := context.Background()
	started, err := service.StartSession(ctx, Identity{Channel: ChannelWebsite, ClientID: "topic-client"})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	sess := started.Session

	if _, err := service.ApplyContextDecision(ctx, &sess, ContextDecision{
		Intent:        "payment_not_found",
		Topic:         "payment",
		LowConfidence: true,
		Event:         EventMessageReceived,
	}); err != nil {
		t.Fatalf("seed payment context: %v", err)
	}
	if _, err := service.ApplyContextDecision(ctx, &sess, ContextDecision{
		Intent: "workspace_availability",
		Topic:  "workspace",
		Event:  EventMessageReceived,
	}); err != nil {
		t.Fatalf("switch topic: %v", err)
	}

	if sess.ActiveTopic != "workspace" || sess.LastIntent != "workspace_availability" || sess.FallbackCount != 0 {
		t.Fatalf("topic switch context mismatch: topic=%q intent=%q fallback=%d",
			sess.ActiveTopic,
			sess.LastIntent,
			sess.FallbackCount)
	}
	if sess.Mode != ModeStandard {
		t.Fatalf("topic switch changed mode to %q, want %q", sess.Mode, ModeStandard)
	}
}

func TestStartSessionRejectsMissingIdentity(t *testing.T) {
	t.Parallel()

	service := NewService(newMemoryRepo())
	_, err := service.StartSession(context.Background(), Identity{Channel: ChannelWebsite})
	if err != ErrInvalidIdentity {
		t.Fatalf("expected ErrInvalidIdentity, got %v", err)
	}
}

func TestStartSessionReturnsExistingSessionWhenCreateRacesWithParallelStarter(t *testing.T) {
	t.Parallel()

	identity := Identity{Channel: ChannelWebsite, ClientID: "racy-client"}
	existing := Session{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		Channel:        identity.Channel,
		ClientID:       identity.ClientID,
		State:          state.StateNew,
		Mode:           ModeStandard,
		OperatorStatus: OperatorStatusNone,
		Status:         StatusActive,
		Metadata:       map[string]interface{}{},
		Version:        1,
	}

	repo := &startRaceRepo{
		activeIdentity: identity,
		activeSession:  existing,
		createErr:      errors.New("duplicate active session"),
	}
	service := NewService(repo)

	result, err := service.StartSession(context.Background(), identity)
	if err != nil {
		t.Fatalf("start session with raced create: %v", err)
	}
	if !result.Resumed {
		t.Fatalf("raced start should resume existing session")
	}
	if result.Session.ID != existing.ID {
		t.Fatalf("resumed wrong session: got %s want %s", result.Session.ID, existing.ID)
	}
	if repo.getActiveCalls != 2 {
		t.Fatalf("getActiveByIdentity calls = %d, want 2", repo.getActiveCalls)
	}
	if repo.createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", repo.createCalls)
	}
}

type memoryRepo struct {
	byID        map[uuid.UUID]Session
	transitions map[uuid.UUID][]ModeTransition
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{
		byID:        make(map[uuid.UUID]Session),
		transitions: make(map[uuid.UUID][]ModeTransition),
	}
}

func (r *memoryRepo) Create(_ context.Context, session Session) (Session, error) {
	normalizeContext(&session)
	if session.Status == "" {
		session.Status = StatusActive
	}
	if session.Version == 0 {
		session.Version = 1
	}
	r.byID[session.ID] = session
	return session, nil
}

func (r *memoryRepo) GetByID(_ context.Context, id uuid.UUID) (Session, error) {
	session, ok := r.byID[id]
	if !ok {
		return Session{}, ErrNotFound
	}
	return session, nil
}

func (r *memoryRepo) GetActiveByIdentity(_ context.Context, identity Identity) (Session, error) {
	identity = NormalizeIdentity(identity)
	for _, session := range r.byID {
		if session.Status != StatusActive || session.Channel != identity.Channel {
			continue
		}
		if identity.ExternalUserID != "" && session.ExternalUserID == identity.ExternalUserID {
			return session, nil
		}
		if identity.ExternalUserID == "" && session.ClientID == identity.ClientID {
			return session, nil
		}
	}
	return Session{}, ErrNotFound
}

func (r *memoryRepo) GetByUserID(context.Context, uuid.UUID, int32, int32) ([]Session, error) {
	return nil, nil
}

func (r *memoryRepo) Update(_ context.Context, session Session) (Session, error) {
	if _, ok := r.byID[session.ID]; !ok {
		return Session{}, ErrNotFound
	}
	normalizeContext(&session)
	session.Version++
	r.byID[session.ID] = session
	return session, nil
}

func (r *memoryRepo) UpdateContext(_ context.Context, session Session, transition *ModeTransition) (Session, error) {
	if _, ok := r.byID[session.ID]; !ok {
		return Session{}, ErrNotFound
	}
	normalizeContext(&session)
	session.Version++
	r.byID[session.ID] = session
	if transition != nil {
		r.transitions[session.ID] = append(r.transitions[session.ID], *transition)
	}
	return session, nil
}

func (r *memoryRepo) UpdateState(_ context.Context, id uuid.UUID, st state.State) (Session, error) {
	session, ok := r.byID[id]
	if !ok {
		return Session{}, ErrNotFound
	}
	session.State = st
	r.byID[id] = session
	return session, nil
}

func (r *memoryRepo) UpdateStateWithVersion(ctx context.Context, id uuid.UUID, st state.State) (Session, error) {
	return r.UpdateState(ctx, id, st)
}

func (r *memoryRepo) UpdateStatus(_ context.Context, id uuid.UUID, status Status) (Session, error) {
	session, ok := r.byID[id]
	if !ok {
		return Session{}, ErrNotFound
	}
	session.Status = status
	r.byID[id] = session
	return session, nil
}

func (r *memoryRepo) List(context.Context, int32, int32) ([]Session, error) {
	return nil, nil
}

func (r *memoryRepo) ListByState(context.Context, state.State, int32, int32) ([]Session, error) {
	return nil, nil
}

func (r *memoryRepo) ListByStatus(context.Context, Status, int32, int32) ([]Session, error) {
	return nil, nil
}

func (r *memoryRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.byID, id)
	return nil
}

func (r *memoryRepo) Count(context.Context) (int64, error) {
	return int64(len(r.byID)), nil
}

type startRaceRepo struct {
	activeIdentity Identity
	activeSession  Session
	createErr      error
	getActiveCalls int
	createCalls    int
}

func (r *startRaceRepo) Create(_ context.Context, session Session) (Session, error) {
	r.createCalls++
	return Session{}, r.createErr
}

func (r *startRaceRepo) GetByID(_ context.Context, id uuid.UUID) (Session, error) {
	if r.activeSession.ID == id {
		return r.activeSession, nil
	}
	return Session{}, ErrNotFound
}

func (r *startRaceRepo) GetActiveByIdentity(_ context.Context, identity Identity) (Session, error) {
	r.getActiveCalls++
	if r.getActiveCalls == 1 {
		return Session{}, ErrNotFound
	}
	if NormalizeIdentity(identity) == NormalizeIdentity(r.activeIdentity) {
		return r.activeSession, nil
	}
	return Session{}, ErrNotFound
}

func (r *startRaceRepo) GetByUserID(context.Context, uuid.UUID, int32, int32) ([]Session, error) {
	return nil, nil
}

func (r *startRaceRepo) Update(_ context.Context, session Session) (Session, error) {
	return session, nil
}

func (r *startRaceRepo) UpdateContext(_ context.Context, session Session, transition *ModeTransition) (Session, error) {
	return session, nil
}

func (r *startRaceRepo) UpdateState(_ context.Context, id uuid.UUID, st state.State) (Session, error) {
	return Session{}, ErrNotFound
}

func (r *startRaceRepo) UpdateStateWithVersion(_ context.Context, id uuid.UUID, st state.State) (Session, error) {
	return Session{}, ErrNotFound
}

func (r *startRaceRepo) UpdateStatus(_ context.Context, id uuid.UUID, status Status) (Session, error) {
	return Session{}, ErrNotFound
}

func (r *startRaceRepo) List(context.Context, int32, int32) ([]Session, error) {
	return nil, nil
}

func (r *startRaceRepo) ListByState(context.Context, state.State, int32, int32) ([]Session, error) {
	return nil, nil
}

func (r *startRaceRepo) ListByStatus(context.Context, Status, int32, int32) ([]Session, error) {
	return nil, nil
}

func (r *startRaceRepo) Delete(context.Context, uuid.UUID) error {
	return nil
}

func (r *startRaceRepo) Count(context.Context) (int64, error) {
	return 1, nil
}

func assertTransition(t *testing.T, got ModeTransition, from Mode, to Mode, event Event) {
	t.Helper()
	if got.From != from || got.To != to || got.Event != event {
		t.Fatalf("transition = %q -> %q on %q, want %q -> %q on %q",
			got.From,
			got.To,
			got.Event,
			from,
			to,
			event)
	}
}
