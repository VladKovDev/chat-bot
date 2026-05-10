package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	operatorDomain "github.com/VladKovDev/chat-bot/internal/domain/operator"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository/postgres/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type operatorRepo struct {
	pool    *Pool
	querier *sqlc.Queries
}

func NewOperatorRepo(pool *Pool) operatorDomain.Repository {
	return &operatorRepo{
		pool:    pool,
		querier: sqlc.New(pool.Pool),
	}
}

func (r *operatorRepo) UpsertOperator(
	ctx context.Context,
	account operatorDomain.Account,
) (operatorDomain.Account, error) {
	dbOperator, err := r.querier.UpsertOperator(ctx, sqlc.UpsertOperatorParams{
		OperatorID:  account.OperatorID,
		FixtureID:   account.FixtureID,
		DisplayName: account.DisplayName,
		Status:      account.Status,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return operatorDomain.Account{}, err
		}
		return operatorDomain.Account{}, fmt.Errorf("failed to upsert operator: %w", err)
	}
	return domainOperatorAccountFromDB(dbOperator), nil
}

func (r *operatorRepo) Queue(
	ctx context.Context,
	req operatorDomain.QueueRequest,
	sessionUpdate session.Session,
	transition *session.ModeTransition,
) (operatorDomain.QueueItem, error) {
	if err := operatorDomain.ValidateReason(req.Reason); err != nil {
		return operatorDomain.QueueItem{}, err
	}
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}

	snapshot, err := marshalContextSnapshot(req.ContextSnapshot)
	if err != nil {
		return operatorDomain.QueueItem{}, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return operatorDomain.QueueItem{}, fmt.Errorf("failed to begin operator queue transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.querier.WithTx(tx)
	dbQueue, err := qtx.CreateOperatorQueueItem(ctx, sqlc.CreateOperatorQueueItemParams{
		ID:              uuidToPgUUID(req.ID),
		SessionID:       uuidToPgUUID(req.SessionID),
		UserID:          uuidToPgUUID(req.UserID),
		Reason:          string(req.Reason),
		Priority:        int32(req.Priority),
		ContextSnapshot: snapshot,
	})
	if err != nil {
		return operatorDomain.QueueItem{}, mapOperatorWriteError(err, "create operator queue item")
	}

	eventPayload, err := json.Marshal(map[string]interface{}{
		"reason": string(req.Reason),
	})
	if err != nil {
		return operatorDomain.QueueItem{}, fmt.Errorf("failed to marshal operator queued event: %w", err)
	}
	if _, err := qtx.CreateOperatorEvent(ctx, sqlc.CreateOperatorEventParams{
		QueueID:   dbQueue.ID,
		SessionID: dbQueue.SessionID,
		EventType: string(operatorDomain.EventQueued),
		ActorType: string(operatorDomain.ActorUser),
		ActorID:   req.UserID.String(),
		Payload:   eventPayload,
	}); err != nil {
		return operatorDomain.QueueItem{}, mapOperatorWriteError(err, "create operator queued event")
	}

	if err := updateSessionContextInOperatorTx(ctx, qtx, sessionUpdate, transition); err != nil {
		return operatorDomain.QueueItem{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return operatorDomain.QueueItem{}, fmt.Errorf("failed to commit operator queue transaction: %w", err)
	}
	return domainOperatorQueueFromDB(dbQueue), nil
}

func (r *operatorRepo) Accept(
	ctx context.Context,
	req operatorDomain.AcceptRequest,
	sessionUpdate session.Session,
	transition *session.ModeTransition,
) (operatorDomain.QueueItem, error) {
	if req.OperatorID == "" {
		return operatorDomain.QueueItem{}, operatorDomain.ErrInvalidOperator
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return operatorDomain.QueueItem{}, fmt.Errorf("failed to begin operator accept transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.querier.WithTx(tx)
	dbQueue, err := qtx.AcceptOperatorQueueItem(ctx, sqlc.AcceptOperatorQueueItemParams{
		ID:         uuidToPgUUID(req.QueueID),
		OperatorID: req.OperatorID,
	})
	if err != nil {
		return operatorDomain.QueueItem{}, mapOperatorTransitionError(err, "accept operator queue item")
	}
	if _, err := qtx.CreateOperatorAssignment(ctx, sqlc.CreateOperatorAssignmentParams{
		QueueID:    dbQueue.ID,
		OperatorID: req.OperatorID,
	}); err != nil {
		return operatorDomain.QueueItem{}, mapOperatorWriteError(err, "create operator assignment")
	}

	eventPayload, err := json.Marshal(map[string]interface{}{
		"operator_id": req.OperatorID,
	})
	if err != nil {
		return operatorDomain.QueueItem{}, fmt.Errorf("failed to marshal operator accepted event: %w", err)
	}
	if _, err := qtx.CreateOperatorEvent(ctx, sqlc.CreateOperatorEventParams{
		QueueID:   dbQueue.ID,
		SessionID: dbQueue.SessionID,
		EventType: string(operatorDomain.EventAccepted),
		ActorType: string(operatorDomain.ActorOperator),
		ActorID:   req.OperatorID,
		Payload:   eventPayload,
	}); err != nil {
		return operatorDomain.QueueItem{}, mapOperatorWriteError(err, "create operator accepted event")
	}

	if err := updateSessionContextInOperatorTx(ctx, qtx, sessionUpdate, transition); err != nil {
		return operatorDomain.QueueItem{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return operatorDomain.QueueItem{}, fmt.Errorf("failed to commit operator accept transaction: %w", err)
	}
	return domainOperatorQueueFromDB(dbQueue), nil
}

func (r *operatorRepo) Close(
	ctx context.Context,
	req operatorDomain.CloseRequest,
	sessionUpdate session.Session,
	transition *session.ModeTransition,
) (operatorDomain.QueueItem, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return operatorDomain.QueueItem{}, fmt.Errorf("failed to begin operator close transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.querier.WithTx(tx)
	dbQueue, err := qtx.CloseOperatorQueueItem(ctx, uuidToPgUUID(req.QueueID))
	if err != nil {
		return operatorDomain.QueueItem{}, mapOperatorTransitionError(err, "close operator queue item")
	}
	if err := qtx.CloseOperatorAssignment(ctx, dbQueue.ID); err != nil {
		return operatorDomain.QueueItem{}, mapOperatorWriteError(err, "close operator assignment")
	}

	actorType := operatorDomain.ActorSystem
	if req.OperatorID != "" {
		actorType = operatorDomain.ActorOperator
	}
	eventPayload, err := json.Marshal(map[string]interface{}{
		"operator_id": req.OperatorID,
	})
	if err != nil {
		return operatorDomain.QueueItem{}, fmt.Errorf("failed to marshal operator closed event: %w", err)
	}
	if _, err := qtx.CreateOperatorEvent(ctx, sqlc.CreateOperatorEventParams{
		QueueID:   dbQueue.ID,
		SessionID: dbQueue.SessionID,
		EventType: string(operatorDomain.EventClosed),
		ActorType: string(actorType),
		ActorID:   req.OperatorID,
		Payload:   eventPayload,
	}); err != nil {
		return operatorDomain.QueueItem{}, mapOperatorWriteError(err, "create operator closed event")
	}

	if err := updateSessionContextInOperatorTx(ctx, qtx, sessionUpdate, transition); err != nil {
		return operatorDomain.QueueItem{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return operatorDomain.QueueItem{}, fmt.Errorf("failed to commit operator close transaction: %w", err)
	}
	return domainOperatorQueueFromDB(dbQueue), nil
}

func (r *operatorRepo) GetByID(ctx context.Context, id uuid.UUID) (operatorDomain.QueueItem, error) {
	dbQueue, err := r.querier.GetOperatorQueueByID(ctx, uuidToPgUUID(id))
	if err != nil {
		return operatorDomain.QueueItem{}, mapOperatorReadError(err)
	}
	return domainOperatorQueueFromDB(dbQueue), nil
}

func (r *operatorRepo) GetOpenBySession(
	ctx context.Context,
	sessionID uuid.UUID,
) (operatorDomain.QueueItem, error) {
	dbQueue, err := r.querier.GetOpenOperatorQueueBySession(ctx, uuidToPgUUID(sessionID))
	if err != nil {
		return operatorDomain.QueueItem{}, mapOperatorReadError(err)
	}
	return domainOperatorQueueFromDB(dbQueue), nil
}

func (r *operatorRepo) ListByStatus(
	ctx context.Context,
	status operatorDomain.QueueStatus,
	limit int32,
	offset int32,
) ([]operatorDomain.QueueItem, error) {
	status = operatorDomain.NormalizeStatus(status)
	if err := operatorDomain.ValidateStatus(status); err != nil {
		return nil, err
	}
	dbQueues, err := r.querier.ListOperatorQueueByStatus(ctx, sqlc.ListOperatorQueueByStatusParams{
		Column1: string(status),
		Column2: limit,
		Column3: offset,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to list operator queue by status: %w", err)
	}
	return domainOperatorQueuesFromDB(dbQueues), nil
}

func marshalContextSnapshot(snapshot operatorDomain.ContextSnapshot) ([]byte, error) {
	if snapshot.LastMessages == nil {
		snapshot.LastMessages = []operatorDomain.MessageSnapshot{}
	}
	if snapshot.ActionSummaries == nil {
		snapshot.ActionSummaries = []operatorDomain.ActionSummary{}
	}
	return json.Marshal(snapshot)
}

func updateSessionContextInOperatorTx(
	ctx context.Context,
	qtx *sqlc.Queries,
	sessionUpdate session.Session,
	transition *session.ModeTransition,
) error {
	metadata, err := marshalMetadata(sessionUpdate.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal session metadata: %w", err)
	}

	if _, err := qtx.UpdateSession(ctx, sqlc.UpdateSessionParams{
		ID:             uuidToPgUUID(sessionUpdate.ID),
		State:          string(sessionUpdate.State),
		Mode:           string(defaultMode(sessionUpdate.Mode)),
		ActiveTopic:    sessionUpdate.ActiveTopic,
		LastIntent:     sessionUpdate.LastIntent,
		FallbackCount:  int32(sessionUpdate.FallbackCount),
		OperatorStatus: string(defaultOperatorStatus(sessionUpdate.OperatorStatus, sessionUpdate.Mode)),
		Metadata:       metadata,
		Status:         string(defaultStatus(sessionUpdate.Status)),
	}); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return fmt.Errorf("failed to update session context in operator transaction: %w", err)
	}

	if transition != nil {
		if _, err := qtx.LogTransition(ctx, sqlc.LogTransitionParams{
			Column1: uuidToPgUUID(transition.SessionID),
			Column2: string(transition.From),
			Column3: string(transition.To),
			Column4: string(transition.Event),
			Column5: transition.Reason,
		}); err != nil {
			return fmt.Errorf("failed to log mode transition in operator transaction: %w", err)
		}
	}

	return nil
}

func mapOperatorReadError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return operatorDomain.ErrNotFound
	}
	return fmt.Errorf("failed to read operator queue: %w", err)
}

func mapOperatorTransitionError(err error, op string) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return operatorDomain.ErrInvalidTransition
	}
	return mapOperatorWriteError(err, op)
}

func mapOperatorWriteError(err error, op string) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return operatorDomain.ErrInvalidOperator
		case "23505":
			return operatorDomain.ErrInvalidTransition
		}
	}
	return fmt.Errorf("failed to %s: %w", op, err)
}

var _ operatorDomain.Repository = (*operatorRepo)(nil)
