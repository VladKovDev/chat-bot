package session

import (
	"context"
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
	if firstA.Session.ChatID == firstB.Session.ChatID {
		t.Fatalf("different browser clients got the same derived chat_id: %d", firstA.Session.ChatID)
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

func TestStartSessionRejectsMissingIdentity(t *testing.T) {
	t.Parallel()

	service := NewService(newMemoryRepo())
	_, err := service.StartSession(context.Background(), Identity{Channel: ChannelWebsite})
	if err != ErrInvalidIdentity {
		t.Fatalf("expected ErrInvalidIdentity, got %v", err)
	}
}

type memoryRepo struct {
	byID map[uuid.UUID]Session
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{byID: make(map[uuid.UUID]Session)}
}

func (r *memoryRepo) Create(_ context.Context, session Session) (Session, error) {
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

func (r *memoryRepo) GetByChatID(_ context.Context, chatID int64) (Session, error) {
	for _, session := range r.byID {
		if session.ChatID == chatID {
			return session, nil
		}
	}
	return Session{}, ErrNotFound
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
	r.byID[session.ID] = session
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

func (r *memoryRepo) UpdateSummary(context.Context, uuid.UUID, string) (Session, error) {
	return Session{}, nil
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
