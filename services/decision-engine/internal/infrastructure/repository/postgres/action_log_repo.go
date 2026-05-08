package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository/postgres/sqlc"
	"github.com/google/uuid"
)

type actionLogRepo struct {
	pool    *Pool
	querier *sqlc.Queries
}

func NewActionLogRepo(pool *Pool) action.LogRepository {
	return &actionLogRepo{
		pool:    pool,
		querier: sqlc.New(pool.Pool),
	}
}

func (r *actionLogRepo) Log(ctx context.Context, entry action.Log) (action.Log, error) {
	var requestPayload []byte
	var responsePayload []byte
	var errorStr string

	if entry.RequestPayload != nil {
		requestPayload, _ = json.Marshal(entry.RequestPayload)
	}

	if entry.ResponsePayload != nil {
		responsePayload, _ = json.Marshal(entry.ResponsePayload)
	}

	if entry.Error != nil {
		errorStr = *entry.Error
	}

	dbLog, err := r.querier.LogAction(ctx, sqlc.LogActionParams{
		Column1: uuidToPgUUID(entry.SessionID),
		Column2: entry.ActionType,
		Column3: requestPayload,
		Column4: responsePayload,
		Column5: errorStr,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return action.Log{}, err
		}
		return action.Log{}, fmt.Errorf("failed to log action: %w", err)
	}

	return domainActionLogFromDB(dbLog), nil
}

func (r *actionLogRepo) GetBySessionID(ctx context.Context, sessionID uuid.UUID, limit int32, offset int32) ([]action.Log, error) {
	dbLogs, err := r.querier.GetActionsBySessionID(ctx, sqlc.GetActionsBySessionIDParams{
		Column1: uuidToPgUUID(sessionID),
		Column2: limit,
		Column3: offset,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get actions by session ID: %w", err)
	}

	return domainActionLogsFromDB(dbLogs), nil
}

func (r *actionLogRepo) GetByType(ctx context.Context, actionType string, limit int32, offset int32) ([]action.Log, error) {
	dbLogs, err := r.querier.GetActionsByType(ctx, sqlc.GetActionsByTypeParams{
		Column1: actionType,
		Column2: limit,
		Column3: offset,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get actions by type: %w", err)
	}

	return domainActionLogsFromDB(dbLogs), nil
}

func (r *actionLogRepo) CountBySessionID(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	count, err := r.querier.CountActions(ctx, uuidToPgUUID(sessionID))
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return 0, err
		}
		return 0, fmt.Errorf("failed to count actions: %w", err)
	}

	return count, nil
}
