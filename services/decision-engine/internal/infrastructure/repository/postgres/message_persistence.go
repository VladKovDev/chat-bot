package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/app/worker"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/message"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository/postgres/sqlc"
	"github.com/google/uuid"
)

type messagePersistence struct {
	pool    *Pool
	querier *sqlc.Queries
}

func NewMessagePersistence(pool *Pool) worker.MessagePersistence {
	return &messagePersistence{
		pool:    pool,
		querier: sqlc.New(pool.Pool),
	}
}

func (p *messagePersistence) WithinMessageTransaction(
	ctx context.Context,
	fn func(context.Context, worker.MessageTransaction) error,
) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin message transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := p.querier.WithTx(tx)
	if err := fn(ctx, &messageTx{queries: qtx}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit message transaction: %w", err)
	}
	return nil
}

type messageTx struct {
	queries *sqlc.Queries
}

func (tx *messageTx) CreateMessage(ctx context.Context, msg message.Message) (message.Message, error) {
	var intentPtr string
	if msg.Intent != nil {
		intentPtr = *msg.Intent
	}

	dbMsg, err := tx.queries.CreateMessage(ctx, sqlc.CreateMessageParams{
		Column1: uuidToPgUUID(msg.SessionID),
		Column2: string(msg.SenderType),
		Column3: msg.Text,
		Column4: intentPtr,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return message.Message{}, err
		}
		return message.Message{}, fmt.Errorf("failed to create message: %w", err)
	}

	return domainMessageFromDB(dbMsg), nil
}

func (tx *messageTx) GetLastMessagesBySessionID(
	ctx context.Context,
	sessionID uuid.UUID,
	limit int32,
) ([]message.Message, error) {
	dbMessages, err := tx.queries.GetLastMessagesBySessionID(ctx, sqlc.GetLastMessagesBySessionIDParams{
		Column1: uuidToPgUUID(sessionID),
		Column2: limit,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get last messages by session ID: %w", err)
	}

	return domainMessagesFromDB(dbMessages), nil
}

func (tx *messageTx) LogDecision(ctx context.Context, entry worker.DecisionLog) error {
	candidates, err := json.Marshal(entry.Candidates)
	if err != nil {
		return fmt.Errorf("failed to marshal decision candidates: %w", err)
	}

	dbLog, err := tx.queries.LogDecision(ctx, sqlc.LogDecisionParams{
		SessionID:      uuidToPgUUID(entry.SessionID),
		MessageID:      uuidToPgUUID(entry.MessageID),
		Intent:         entry.Intent,
		State:          string(entry.State),
		ResponseKey:    entry.ResponseKey,
		Confidence:     entry.Confidence,
		LowConfidence:  entry.LowConfidence,
		FallbackReason: entry.FallbackReason,
		Threshold:      entry.Threshold,
		Candidates:     candidates,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return fmt.Errorf("failed to log decision: %w", err)
	}
	for index, candidate := range entry.Candidates {
		if candidate.IntentKey == "" {
			continue
		}
		source := candidate.Source
		if source == "" {
			source = "intent_example"
		}
		if !allowedCandidateSource(source) {
			return fmt.Errorf("unsupported decision candidate source: %s", source)
		}
		metadata, err := json.Marshal(map[string]any{
			"text":     candidate.Text,
			"metadata": candidate.Metadata,
		})
		if err != nil {
			return fmt.Errorf("failed to marshal decision candidate metadata: %w", err)
		}
		if _, err := tx.queries.LogDecisionCandidate(ctx, sqlc.LogDecisionCandidateParams{
			DecisionLogID: dbLog.ID,
			IntentID:      nullablePgUUIDFromString(candidate.IntentID),
			IntentKey:     candidate.IntentKey,
			Confidence:    candidate.Confidence,
			Rank:          int32(index + 1),
			Source:        source,
			Metadata:      metadata,
		}); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			return fmt.Errorf("failed to log decision candidate: %w", err)
		}
	}
	return nil
}

func allowedCandidateSource(source string) bool {
	switch source {
	case "intent_example", "knowledge_chunk", "exact_command", "fallback", "lexical_fuzzy", "quick_reply_intent", "contextual_rule":
		return true
	default:
		return false
	}
}

func (tx *messageTx) LogAction(ctx context.Context, entry action.Log) error {
	var requestPayload []byte
	var responsePayload []byte
	var errorStr string
	var err error

	if entry.RequestPayload != nil {
		requestPayload, err = json.Marshal(entry.RequestPayload)
		if err != nil {
			return fmt.Errorf("failed to marshal action request payload: %w", err)
		}
	}

	if entry.ResponsePayload != nil {
		responsePayload, err = json.Marshal(entry.ResponsePayload)
		if err != nil {
			return fmt.Errorf("failed to marshal action response payload: %w", err)
		}
	}

	if entry.Error != nil {
		errorStr = *entry.Error
	}

	if _, err := tx.queries.LogAction(ctx, sqlc.LogActionParams{
		Column1: uuidToPgUUID(entry.SessionID),
		Column2: entry.ActionType,
		Column3: requestPayload,
		Column4: responsePayload,
		Column5: errorStr,
	}); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return fmt.Errorf("failed to log action: %w", err)
	}
	return nil
}

func (tx *messageTx) ApplyContextDecision(
	ctx context.Context,
	sess *session.Session,
	decision session.ContextDecision,
) (session.Session, error) {
	next, transition, err := session.PrepareContextUpdate(sess, decision)
	if err != nil {
		return session.Session{}, err
	}

	metadata, err := marshalMetadata(next.Metadata)
	if err != nil {
		return session.Session{}, fmt.Errorf("failed to marshal session metadata: %w", err)
	}

	dbSession, err := tx.queries.UpdateSession(ctx, sqlc.UpdateSessionParams{
		ID:             uuidToPgUUID(next.ID),
		State:          string(next.State),
		Mode:           string(defaultMode(next.Mode)),
		ActiveTopic:    next.ActiveTopic,
		LastIntent:     next.LastIntent,
		FallbackCount:  int32(next.FallbackCount),
		OperatorStatus: string(defaultOperatorStatus(next.OperatorStatus, next.Mode)),
		Metadata:       metadata,
		Status:         string(defaultStatus(next.Status)),
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return session.Session{}, err
		}
		return session.Session{}, session.ErrNotFound
	}

	if transition != nil {
		if _, err := tx.queries.LogTransition(ctx, sqlc.LogTransitionParams{
			Column1: uuidToPgUUID(transition.SessionID),
			Column2: string(transition.From),
			Column3: string(transition.To),
			Column4: string(transition.Event),
			Column5: transition.Reason,
		}); err != nil {
			return session.Session{}, fmt.Errorf("failed to log mode transition: %w", err)
		}
	}

	updated := domainSessionFromDB(dbSession)
	*sess = updated
	return updated, nil
}

var _ worker.MessagePersistence = (*messagePersistence)(nil)

var _ worker.MessageTransaction = (*messageTx)(nil)
