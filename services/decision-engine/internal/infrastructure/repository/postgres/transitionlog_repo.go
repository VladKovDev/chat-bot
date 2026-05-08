package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/transitionlog"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository/postgres/sqlc"
	"github.com/google/uuid"
)

type transitionLogRepo struct {
	pool    *Pool
	querier *sqlc.Queries
}

func NewTransitionLogRepo(pool *Pool) transitionlog.Repository {
	return &transitionLogRepo{
		pool:    pool,
		querier: sqlc.New(pool.Pool),
	}
}

func (r *transitionLogRepo) Log(ctx context.Context, entry transitionlog.TransitionLog) (transitionlog.TransitionLog, error) {
	dbLog, err := r.querier.LogTransition(ctx, sqlc.LogTransitionParams{
		Column1: uuidToPgUUID(entry.SessionID),
		Column2: string(entry.FromState),
		Column3: string(entry.ToState),
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return transitionlog.TransitionLog{}, err
		}
		return transitionlog.TransitionLog{}, fmt.Errorf("failed to log transition: %w", err)
	}

	return domainTransitionLogFromDB(dbLog), nil
}

func (r *transitionLogRepo) GetBySessionID(ctx context.Context, sessionID uuid.UUID, limit int32, offset int32) ([]transitionlog.TransitionLog, error) {
	dbLogs, err := r.querier.GetTransitionsBySessionID(ctx, sqlc.GetTransitionsBySessionIDParams{
		Column1: uuidToPgUUID(sessionID),
		Column2: limit,
		Column3: offset,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get transitions by session ID: %w", err)
	}

	return domainTransitionLogsFromDB(dbLogs), nil
}

func (r *transitionLogRepo) CountBySessionID(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	count, err := r.querier.CountTransitions(ctx, uuidToPgUUID(sessionID))
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return 0, err
		}
		return 0, fmt.Errorf("failed to count transitions: %w", err)
	}

	return count, nil
}
