package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository/postgres/sqlc"
	"github.com/google/uuid"
)

type sessionRepo struct {
	pool    *Pool
	querier *sqlc.Queries
}

func NewSessionRepo(pool *Pool) session.Repository {
	return &sessionRepo{
		pool:    pool,
		querier: sqlc.New(pool.Pool),
	}
}

func (r *sessionRepo) Create(ctx context.Context, s session.Session) (session.Session, error) {
	metadata, err := marshalMetadata(s.Metadata)
	if err != nil {
		return session.Session{}, fmt.Errorf("failed to marshal session metadata: %w", err)
	}

	dbSession, err := r.querier.CreateSession(ctx, sqlc.CreateSessionParams{
		ChatID:         s.ChatID,
		UserID:         uuidToPgUUID(s.UserID),
		Channel:        s.Channel,
		ExternalUserID: s.ExternalUserID,
		ClientID:       s.ClientID,
		State:          string(s.State),
		Mode:           string(defaultMode(s.Mode)),
		ActiveTopic:    s.ActiveTopic,
		LastIntent:     s.LastIntent,
		FallbackCount:  int32(s.FallbackCount),
		OperatorStatus: string(defaultOperatorStatus(s.OperatorStatus, s.Mode)),
		Metadata:       metadata,
	})
	if err != nil {
		return session.Session{}, fmt.Errorf("failed to create session: %w", err)
	}

	return domainSessionFromDB(dbSession), nil
}

func (r *sessionRepo) GetByID(ctx context.Context, id uuid.UUID) (session.Session, error) {
	dbSession, err := r.querier.GetSessionByID(ctx, uuidToPgUUID(id))
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return session.Session{}, err
		}
		return session.Session{}, session.ErrNotFound
	}

	return domainSessionFromDB(dbSession), nil
}

func (r *sessionRepo) GetByChatID(
	ctx context.Context,
	chatID int64,
) (session.Session, error) {
	dbSession, err := r.querier.GetSessionByChatID(ctx, chatID)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return session.Session{}, err
		}
		return session.Session{}, session.ErrNotFound
	}

	return domainSessionFromDB(dbSession), nil
}

func (r *sessionRepo) GetActiveByIdentity(
	ctx context.Context,
	identity session.Identity,
) (session.Session, error) {
	dbSession, err := r.querier.GetActiveSessionByIdentity(ctx, sqlc.GetActiveSessionByIdentityParams{
		Channel:        identity.Channel,
		ExternalUserID: identity.ExternalUserID,
		ClientID:       identity.ClientID,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return session.Session{}, err
		}
		return session.Session{}, session.ErrNotFound
	}

	return domainSessionFromDB(dbSession), nil
}

func (r *sessionRepo) GetByUserID(
	ctx context.Context,
	userID uuid.UUID,
	limit int32,
	offset int32,
) ([]session.Session, error) {
	dbSessions, err := r.querier.ListSessionsByUser(ctx, sqlc.ListSessionsByUserParams{
		Column1: uuidToPgUUID(userID),
		Column2: limit,
		Column3: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions by user: %w", err)
	}

	return domainSessionsFromDB(dbSessions), nil
}

func (r *sessionRepo) Update(
	ctx context.Context,
	s session.Session,
) (session.Session, error) {
	metadata, err := marshalMetadata(s.Metadata)
	if err != nil {
		return session.Session{}, fmt.Errorf("failed to marshal session metadata: %w", err)
	}

	dbSession, err := r.querier.UpdateSession(ctx, sqlc.UpdateSessionParams{
		ID:             uuidToPgUUID(s.ID),
		State:          string(s.State),
		Mode:           string(defaultMode(s.Mode)),
		ActiveTopic:    s.ActiveTopic,
		LastIntent:     s.LastIntent,
		FallbackCount:  int32(s.FallbackCount),
		OperatorStatus: string(defaultOperatorStatus(s.OperatorStatus, s.Mode)),
		Metadata:       metadata,
		Status:         string(defaultStatus(s.Status)),
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return session.Session{}, err
		}
		return session.Session{}, session.ErrNotFound
	}

	return domainSessionFromDB(dbSession), nil
}

func (r *sessionRepo) UpdateContext(
	ctx context.Context,
	s session.Session,
	transition *session.ModeTransition,
) (session.Session, error) {
	metadata, err := marshalMetadata(s.Metadata)
	if err != nil {
		return session.Session{}, fmt.Errorf("failed to marshal session metadata: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return session.Session{}, fmt.Errorf("failed to begin session context transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.querier.WithTx(tx)
	dbSession, err := qtx.UpdateSession(ctx, sqlc.UpdateSessionParams{
		ID:             uuidToPgUUID(s.ID),
		State:          string(s.State),
		Mode:           string(defaultMode(s.Mode)),
		ActiveTopic:    s.ActiveTopic,
		LastIntent:     s.LastIntent,
		FallbackCount:  int32(s.FallbackCount),
		OperatorStatus: string(defaultOperatorStatus(s.OperatorStatus, s.Mode)),
		Metadata:       metadata,
		Status:         string(defaultStatus(s.Status)),
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return session.Session{}, err
		}
		return session.Session{}, session.ErrNotFound
	}

	if transition != nil {
		if _, err := qtx.LogTransition(ctx, sqlc.LogTransitionParams{
			Column1: uuidToPgUUID(transition.SessionID),
			Column2: string(transition.From),
			Column3: string(transition.To),
			Column4: string(transition.Event),
			Column5: transition.Reason,
		}); err != nil {
			return session.Session{}, fmt.Errorf("failed to log mode transition: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return session.Session{}, fmt.Errorf("failed to commit session context transaction: %w", err)
	}

	return domainSessionFromDB(dbSession), nil
}

func (r *sessionRepo) UpdateState(
	ctx context.Context,
	id uuid.UUID,
	st state.State,
) (session.Session, error) {
	dbSession, err := r.querier.UpdateSessionState(ctx, sqlc.UpdateSessionStateParams{
		Column1: uuidToPgUUID(id),
		Column2: string(st),
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return session.Session{}, err
		}
		return session.Session{}, session.ErrNotFound
	}

	return domainSessionFromDB(dbSession), nil
}

func (r *sessionRepo) UpdateStateWithVersion(
	ctx context.Context,
	id uuid.UUID,
	st state.State,
) (session.Session, error) {
	dbSession, err := r.querier.UpdateSessionWithVersion(ctx, sqlc.UpdateSessionWithVersionParams{
		Column1: uuidToPgUUID(id),
		Column2: string(st),
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return session.Session{}, err
		}
		return session.Session{}, session.ErrNotFound
	}

	return domainSessionFromDB(dbSession), nil
}

func (r *sessionRepo) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status session.Status,
) (session.Session, error) {
	dbSession, err := r.querier.UpdateSessionStatus(ctx, sqlc.UpdateSessionStatusParams{
		Column1: uuidToPgUUID(id),
		Column2: string(status),
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return session.Session{}, err
		}
		return session.Session{}, session.ErrNotFound
	}

	return domainSessionFromDB(dbSession), nil
}

func (r *sessionRepo) UpdateSummary(
	ctx context.Context,
	id uuid.UUID,
	summary string,
) (session.Session, error) {
	dbSession, err := r.querier.UpdateSessionSummary(ctx, sqlc.UpdateSessionSummaryParams{
		Column1: uuidToPgUUID(id),
		Column2: summary,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return session.Session{}, err
		}
		return session.Session{}, session.ErrNotFound
	}

	return domainSessionFromDB(dbSession), nil
}

func (r *sessionRepo) List(
	ctx context.Context,
	limit int32,
	offset int32,
) ([]session.Session, error) {
	dbSessions, err := r.querier.ListSessions(ctx, sqlc.ListSessionsParams{
		Column1: limit,
		Column2: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	return domainSessionsFromDB(dbSessions), nil
}

func (r *sessionRepo) ListByState(
	ctx context.Context,
	st state.State,
	limit int32,
	offset int32,
) ([]session.Session, error) {
	dbSessions, err := r.querier.ListSessionsByState(ctx, sqlc.ListSessionsByStateParams{
		Column1: string(st),
		Column2: limit,
		Column3: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions by state: %w", err)
	}

	return domainSessionsFromDB(dbSessions), nil
}

func (r *sessionRepo) ListByStatus(
	ctx context.Context,
	status session.Status,
	limit int32,
	offset int32,
) ([]session.Session, error) {
	dbSessions, err := r.querier.ListSessionsByStatus(ctx, sqlc.ListSessionsByStatusParams{
		Column1: string(status),
		Column2: limit,
		Column3: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions by status: %w", err)
	}

	return domainSessionsFromDB(dbSessions), nil
}

func (r *sessionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.querier.DeleteSession(ctx, uuidToPgUUID(id))
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return session.ErrNotFound
	}
	return nil
}

func (r *sessionRepo) Count(ctx context.Context) (int64, error) {
	count, err := r.querier.CountSessions(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count sessions: %w", err)
	}
	return count, nil
}

func defaultMode(mode session.Mode) session.Mode {
	if mode == "" {
		return session.ModeStandard
	}
	return mode
}

func defaultOperatorStatus(status session.OperatorStatus, mode session.Mode) session.OperatorStatus {
	if status != "" {
		return status
	}
	switch defaultMode(mode) {
	case session.ModeWaitingOperator:
		return session.OperatorStatusWaiting
	case session.ModeOperatorConnected:
		return session.OperatorStatusConnected
	case session.ModeClosed:
		return session.OperatorStatusClosed
	default:
		return session.OperatorStatusNone
	}
}

func defaultStatus(status session.Status) session.Status {
	if status == "" {
		return session.StatusActive
	}
	return status
}

func marshalMetadata(metadata map[string]interface{}) ([]byte, error) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	return json.Marshal(metadata)
}
