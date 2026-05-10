package operator

import (
	"context"
	"errors"
	"testing"
	"time"

	operatorDomain "github.com/VladKovDev/chat-bot/internal/domain/operator"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/google/uuid"
)

var testNow = time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

func TestServiceQueuesAcceptsAndClosesHandoff(t *testing.T) {
	t.Parallel()

	sessionID := uuid.New()
	userID := uuid.New()
	sessions := newFakeSessions(session.Session{
		ID:             sessionID,
		UserID:         userID,
		State:          state.StatePayment,
		Mode:           session.ModeStandard,
		Status:         session.StatusActive,
		OperatorStatus: session.OperatorStatusNone,
		ActiveTopic:    "payment",
		LastIntent:     "payment_not_activated",
		Metadata:       map[string]interface{}{},
	})
	queue := newFakeQueue(sessions)
	service := NewService(queue, sessions)

	snapshot := operatorDomain.ContextSnapshot{
		LastMessages: []operatorDomain.MessageSnapshot{
			{SenderType: "user", Text: "Деньги списались", CreatedAt: testNow.Add(-2 * time.Minute)},
			{SenderType: "bot", Text: "Проверяю оплату", Intent: "ask_payment_status", CreatedAt: testNow.Add(-time.Minute)},
		},
		ActiveTopic: "payment",
		LastIntent:  "payment_not_activated",
		ActionSummaries: []operatorDomain.ActionSummary{
			{ActionType: "find_payment", Status: "business_error", Summary: "provider unavailable", CreatedAt: testNow},
		},
	}

	queued, err := service.Queue(context.Background(), sessionID, operatorDomain.ReasonBusinessError, snapshot)
	if err != nil {
		t.Fatalf("queue handoff: %v", err)
	}
	if queued.Status != operatorDomain.QueueStatusWaiting {
		t.Fatalf("queued status = %q, want waiting", queued.Status)
	}
	if queued.Reason != operatorDomain.ReasonBusinessError {
		t.Fatalf("queued reason = %q, want business_error", queued.Reason)
	}
	if queued.ContextSnapshot.LastIntent != snapshot.LastIntent || len(queued.ContextSnapshot.ActionSummaries) != 1 {
		t.Fatalf("snapshot not preserved: %+v", queued.ContextSnapshot)
	}
	if got := sessions.items[sessionID].Mode; got != session.ModeWaitingOperator {
		t.Fatalf("session mode after queue = %q, want %q", got, session.ModeWaitingOperator)
	}

	waiting, err := service.ListByStatus(context.Background(), operatorDomain.QueueStatusWaiting, 100, 0)
	if err != nil {
		t.Fatalf("list waiting: %v", err)
	}
	if len(waiting) != 1 || waiting[0].ID != queued.ID {
		t.Fatalf("waiting queue = %+v, want queued item", waiting)
	}

	accepted, err := service.Accept(context.Background(), queued.ID, "OP-001")
	if err != nil {
		t.Fatalf("accept handoff: %v", err)
	}
	if accepted.Status != operatorDomain.QueueStatusAccepted || accepted.AssignedOperatorID != "OP-001" {
		t.Fatalf("accepted item = %+v", accepted)
	}
	if got := sessions.items[sessionID].Mode; got != session.ModeOperatorConnected {
		t.Fatalf("session mode after accept = %q, want %q", got, session.ModeOperatorConnected)
	}
	if len(queue.assignments) != 1 {
		t.Fatalf("assignments = %+v, want 1", queue.assignments)
	}

	closed, err := service.Close(context.Background(), queued.ID, "OP-001")
	if err != nil {
		t.Fatalf("close handoff: %v", err)
	}
	if closed.Status != operatorDomain.QueueStatusClosed {
		t.Fatalf("closed status = %q, want closed", closed.Status)
	}
	if got := sessions.items[sessionID].Mode; got != session.ModeClosed {
		t.Fatalf("session mode after close = %q, want %q", got, session.ModeClosed)
	}
	if queue.assignments[0].Status != operatorDomain.QueueStatusClosed {
		t.Fatalf("assignment status = %q, want closed", queue.assignments[0].Status)
	}
	if gotEvents := queue.events; len(gotEvents) != 3 ||
		gotEvents[0] != operatorDomain.EventQueued ||
		gotEvents[1] != operatorDomain.EventAccepted ||
		gotEvents[2] != operatorDomain.EventClosed {
		t.Fatalf("events = %+v, want queued/accepted/closed", gotEvents)
	}
}

func TestServiceValidatesReasonAndTransitions(t *testing.T) {
	t.Parallel()

	sessionID := uuid.New()
	sessions := newFakeSessions(session.Session{
		ID:             sessionID,
		UserID:         uuid.New(),
		State:          state.StateNew,
		Mode:           session.ModeStandard,
		Status:         session.StatusActive,
		OperatorStatus: session.OperatorStatusNone,
		Metadata:       map[string]interface{}{},
	})
	queue := newFakeQueue(sessions)
	service := NewService(queue, sessions)

	if _, err := service.Queue(context.Background(), sessionID, operatorDomain.Reason("unexpected"), operatorDomain.ContextSnapshot{}); !errors.Is(err, operatorDomain.ErrInvalidReason) {
		t.Fatalf("invalid reason error = %v, want ErrInvalidReason", err)
	}
	if len(queue.items) != 0 {
		t.Fatalf("queue item was created for invalid reason: %+v", queue.items)
	}

	if _, err := service.Accept(context.Background(), uuid.New(), "OP-001"); !errors.Is(err, operatorDomain.ErrNotFound) {
		t.Fatalf("accept missing handoff error = %v, want ErrNotFound", err)
	}

	queued, err := service.Queue(context.Background(), sessionID, operatorDomain.ReasonManualRequest, operatorDomain.ContextSnapshot{})
	if err != nil {
		t.Fatalf("queue manual handoff: %v", err)
	}
	if _, err := service.Accept(context.Background(), queued.ID, ""); !errors.Is(err, operatorDomain.ErrInvalidOperator) {
		t.Fatalf("empty operator error = %v, want ErrInvalidOperator", err)
	}
	if _, err := service.Close(context.Background(), queued.ID, "OP-001"); err != nil {
		t.Fatalf("close waiting handoff should be allowed: %v", err)
	}
	if _, err := service.Accept(context.Background(), queued.ID, "OP-001"); !errors.Is(err, operatorDomain.ErrInvalidTransition) {
		t.Fatalf("accept closed handoff error = %v, want ErrInvalidTransition", err)
	}
}

func TestServiceQueueWithDecisionPersistsHandoffContext(t *testing.T) {
	t.Parallel()

	sessionID := uuid.New()
	sessions := newFakeSessions(session.Session{
		ID:             sessionID,
		UserID:         uuid.New(),
		State:          state.StateWaitingClarification,
		Mode:           session.ModeStandard,
		Status:         session.StatusActive,
		OperatorStatus: session.OperatorStatusNone,
		FallbackCount:  1,
		Metadata:       map[string]interface{}{},
	})
	queue := newFakeQueue(sessions)
	service := NewService(queue, sessions)

	queued, err := service.QueueWithDecision(
		context.Background(),
		sessionID,
		operatorDomain.ReasonLowConfidenceRepeated,
		operatorDomain.ContextSnapshot{LastIntent: "unknown"},
		session.ContextDecision{
			Intent:        "unknown",
			LowConfidence: true,
			Metadata: map[string]interface{}{
				"decision_response_key": "operator_handoff_requested",
			},
		},
	)
	if err != nil {
		t.Fatalf("queue with decision: %v", err)
	}
	if queued.Reason != operatorDomain.ReasonLowConfidenceRepeated {
		t.Fatalf("reason = %q, want low_confidence_repeated", queued.Reason)
	}

	updated := sessions.items[sessionID]
	if updated.Mode != session.ModeWaitingOperator || updated.OperatorStatus != session.OperatorStatusWaiting {
		t.Fatalf("mode/operator = %q/%q, want waiting_operator/waiting", updated.Mode, updated.OperatorStatus)
	}
	if updated.LastIntent != "unknown" || updated.FallbackCount != 2 {
		t.Fatalf("context = intent:%q fallback:%d, want unknown/2", updated.LastIntent, updated.FallbackCount)
	}
	if updated.Metadata["handoff_reason"] != string(operatorDomain.ReasonLowConfidenceRepeated) ||
		updated.Metadata["decision_response_key"] != "operator_handoff_requested" {
		t.Fatalf("metadata = %#v, want handoff and decision metadata", updated.Metadata)
	}
}

func TestServiceLeavesNoQueueSideEffectWhenAtomicSessionUpdateFails(t *testing.T) {
	t.Parallel()

	sessionID := uuid.New()
	sessions := newFakeSessions(session.Session{
		ID:             sessionID,
		UserID:         uuid.New(),
		State:          state.StateNew,
		Mode:           session.ModeStandard,
		Status:         session.StatusActive,
		OperatorStatus: session.OperatorStatusNone,
		Metadata:       map[string]interface{}{},
	})
	queue := newFakeQueue(sessions)
	queue.failSessionUpdate = errors.New("session update failed")
	service := NewService(queue, sessions)

	if _, err := service.Queue(context.Background(), sessionID, operatorDomain.ReasonManualRequest, operatorDomain.ContextSnapshot{}); !errors.Is(err, queue.failSessionUpdate) {
		t.Fatalf("queue error = %v, want injected session update error", err)
	}
	if len(queue.items) != 0 || len(queue.events) != 0 {
		t.Fatalf("queue side effects leaked after failed session update: items=%+v events=%+v", queue.items, queue.events)
	}
	if got := sessions.items[sessionID].Mode; got != session.ModeStandard {
		t.Fatalf("session mode = %q, want unchanged standard", got)
	}

	queue.failSessionUpdate = nil
	queued, err := service.Queue(context.Background(), sessionID, operatorDomain.ReasonManualRequest, operatorDomain.ContextSnapshot{})
	if err != nil {
		t.Fatalf("queue after clearing failure: %v", err)
	}

	queue.failSessionUpdate = errors.New("accept session update failed")
	if _, err := service.Accept(context.Background(), queued.ID, "OP-001"); !errors.Is(err, queue.failSessionUpdate) {
		t.Fatalf("accept error = %v, want injected session update error", err)
	}
	stored := queue.items[queued.ID]
	if stored.Status != operatorDomain.QueueStatusWaiting || stored.AssignedOperatorID != "" || len(queue.assignments) != 0 {
		t.Fatalf("accept side effects leaked after failed session update: item=%+v assignments=%+v", stored, queue.assignments)
	}
	if len(queue.events) != 1 || queue.events[0] != operatorDomain.EventQueued {
		t.Fatalf("events after failed accept = %+v, want only queued", queue.events)
	}

	queue.failSessionUpdate = nil
	accepted, err := service.Accept(context.Background(), queued.ID, "OP-001")
	if err != nil {
		t.Fatalf("accept after clearing failure: %v", err)
	}

	queue.failSessionUpdate = errors.New("close session update failed")
	if _, err := service.Close(context.Background(), accepted.ID, "OP-001"); !errors.Is(err, queue.failSessionUpdate) {
		t.Fatalf("close error = %v, want injected session update error", err)
	}
	stored = queue.items[accepted.ID]
	if stored.Status != operatorDomain.QueueStatusAccepted || queue.assignments[0].Status != operatorDomain.QueueStatusAccepted {
		t.Fatalf("close side effects leaked after failed session update: item=%+v assignments=%+v", stored, queue.assignments)
	}
	if len(queue.events) != 2 || queue.events[1] != operatorDomain.EventAccepted {
		t.Fatalf("events after failed close = %+v, want queued/accepted only", queue.events)
	}
}

type fakeSessions struct {
	items map[uuid.UUID]session.Session
}

func newFakeSessions(initial session.Session) *fakeSessions {
	return &fakeSessions{items: map[uuid.UUID]session.Session{initial.ID: initial}}
}

func (f *fakeSessions) GetByID(_ context.Context, id uuid.UUID) (session.Session, error) {
	sess, ok := f.items[id]
	if !ok {
		return session.Session{}, session.ErrNotFound
	}
	return sess, nil
}

func (f *fakeSessions) ApplyContextDecision(
	_ context.Context,
	sess *session.Session,
	decision session.ContextDecision,
) (session.Session, error) {
	next, _, err := session.PrepareContextUpdate(sess, decision)
	if err != nil {
		return session.Session{}, err
	}
	f.items[next.ID] = next
	*sess = next
	return next, nil
}

type fakeQueue struct {
	sessions          *fakeSessions
	items             map[uuid.UUID]operatorDomain.QueueItem
	assignments       []operatorDomain.Assignment
	events            []operatorDomain.EventType
	failSessionUpdate error
}

func newFakeQueue(sessions *fakeSessions) *fakeQueue {
	return &fakeQueue{
		sessions: sessions,
		items:    make(map[uuid.UUID]operatorDomain.QueueItem),
	}
}

func (f *fakeQueue) UpsertOperator(_ context.Context, account operatorDomain.Account) (operatorDomain.Account, error) {
	return account, nil
}

func (f *fakeQueue) Queue(
	_ context.Context,
	req operatorDomain.QueueRequest,
	sessionUpdate session.Session,
	_ *session.ModeTransition,
) (operatorDomain.QueueItem, error) {
	for _, item := range f.items {
		if item.SessionID == req.SessionID && item.Status != operatorDomain.QueueStatusClosed {
			return operatorDomain.QueueItem{}, operatorDomain.ErrInvalidTransition
		}
	}
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}
	item := operatorDomain.QueueItem{
		ID:              req.ID,
		SessionID:       req.SessionID,
		UserID:          req.UserID,
		Status:          operatorDomain.QueueStatusWaiting,
		Reason:          req.Reason,
		Priority:        req.Priority,
		ContextSnapshot: req.ContextSnapshot,
		CreatedAt:       testNow,
		UpdatedAt:       testNow,
	}
	previousItems := cloneQueueItems(f.items)
	previousEvents := append([]operatorDomain.EventType(nil), f.events...)
	f.items[item.ID] = item
	f.events = append(f.events, operatorDomain.EventQueued)
	if err := f.applySessionUpdate(sessionUpdate); err != nil {
		f.items = previousItems
		f.events = previousEvents
		return operatorDomain.QueueItem{}, err
	}
	return item, nil
}

func (f *fakeQueue) Accept(
	_ context.Context,
	req operatorDomain.AcceptRequest,
	sessionUpdate session.Session,
	_ *session.ModeTransition,
) (operatorDomain.QueueItem, error) {
	item, ok := f.items[req.QueueID]
	if !ok || item.Status != operatorDomain.QueueStatusWaiting {
		return operatorDomain.QueueItem{}, operatorDomain.ErrInvalidTransition
	}
	previousItems := cloneQueueItems(f.items)
	previousAssignments := append([]operatorDomain.Assignment(nil), f.assignments...)
	previousEvents := append([]operatorDomain.EventType(nil), f.events...)
	item.Status = operatorDomain.QueueStatusAccepted
	item.AssignedOperatorID = req.OperatorID
	item.UpdatedAt = testNow
	acceptedAt := testNow
	item.AcceptedAt = &acceptedAt
	f.items[item.ID] = item
	f.assignments = append(f.assignments, operatorDomain.Assignment{
		ID:         uuid.New(),
		QueueID:    item.ID,
		OperatorID: req.OperatorID,
		Status:     operatorDomain.QueueStatusAccepted,
		AssignedAt: testNow,
	})
	f.events = append(f.events, operatorDomain.EventAccepted)
	if err := f.applySessionUpdate(sessionUpdate); err != nil {
		f.items = previousItems
		f.assignments = previousAssignments
		f.events = previousEvents
		return operatorDomain.QueueItem{}, err
	}
	return item, nil
}

func (f *fakeQueue) Close(
	_ context.Context,
	req operatorDomain.CloseRequest,
	sessionUpdate session.Session,
	_ *session.ModeTransition,
) (operatorDomain.QueueItem, error) {
	item, ok := f.items[req.QueueID]
	if !ok || item.Status == operatorDomain.QueueStatusClosed {
		return operatorDomain.QueueItem{}, operatorDomain.ErrInvalidTransition
	}
	previousItems := cloneQueueItems(f.items)
	previousAssignments := append([]operatorDomain.Assignment(nil), f.assignments...)
	previousEvents := append([]operatorDomain.EventType(nil), f.events...)
	item.Status = operatorDomain.QueueStatusClosed
	item.UpdatedAt = testNow
	closedAt := testNow
	item.ClosedAt = &closedAt
	f.items[item.ID] = item
	for i := range f.assignments {
		if f.assignments[i].QueueID == item.ID && f.assignments[i].Status == operatorDomain.QueueStatusAccepted {
			f.assignments[i].Status = operatorDomain.QueueStatusClosed
			releasedAt := testNow
			f.assignments[i].ReleasedAt = &releasedAt
		}
	}
	f.events = append(f.events, operatorDomain.EventClosed)
	if err := f.applySessionUpdate(sessionUpdate); err != nil {
		f.items = previousItems
		f.assignments = previousAssignments
		f.events = previousEvents
		return operatorDomain.QueueItem{}, err
	}
	return item, nil
}

func (f *fakeQueue) applySessionUpdate(sessionUpdate session.Session) error {
	if f.failSessionUpdate != nil {
		return f.failSessionUpdate
	}
	if sessionUpdate.ID == uuid.Nil {
		return errors.New("missing session update")
	}
	if f.sessions != nil {
		f.sessions.items[sessionUpdate.ID] = sessionUpdate
	}
	return nil
}

func (f *fakeQueue) GetByID(_ context.Context, id uuid.UUID) (operatorDomain.QueueItem, error) {
	item, ok := f.items[id]
	if !ok {
		return operatorDomain.QueueItem{}, operatorDomain.ErrNotFound
	}
	return item, nil
}

func (f *fakeQueue) GetOpenBySession(_ context.Context, sessionID uuid.UUID) (operatorDomain.QueueItem, error) {
	for _, item := range f.items {
		if item.SessionID == sessionID && item.Status != operatorDomain.QueueStatusClosed {
			return item, nil
		}
	}
	return operatorDomain.QueueItem{}, operatorDomain.ErrNotFound
}

func (f *fakeQueue) ListByStatus(_ context.Context, status operatorDomain.QueueStatus, _ int32, _ int32) ([]operatorDomain.QueueItem, error) {
	items := make([]operatorDomain.QueueItem, 0)
	for _, item := range f.items {
		if item.Status == status {
			items = append(items, item)
		}
	}
	return items, nil
}

func cloneQueueItems(items map[uuid.UUID]operatorDomain.QueueItem) map[uuid.UUID]operatorDomain.QueueItem {
	cloned := make(map[uuid.UUID]operatorDomain.QueueItem, len(items))
	for key, value := range items {
		cloned[key] = value
	}
	return cloned
}
