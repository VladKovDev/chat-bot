package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/message"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository/postgres/sqlc"
	"github.com/google/uuid"
)

type messageRepo struct {
	pool    *Pool
	querier *sqlc.Queries
}

func NewMessageRepo(pool *Pool) message.Repository {
	return &messageRepo{
		pool:    pool,
		querier: sqlc.New(pool.Pool),
	}
}

func (r *messageRepo) Create(ctx context.Context, msg message.Message) (message.Message, error) {
	var intentPtr string
	if msg.Intent != nil {
		intentPtr = *msg.Intent
	}

	dbMsg, err := r.querier.CreateMessage(ctx, sqlc.CreateMessageParams{
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

func (r *messageRepo) GetBySessionID(ctx context.Context, sessionID uuid.UUID, limit int32, offset int32) ([]message.Message, error) {
	dbMessages, err := r.querier.GetMessagesBySessionID(ctx, sqlc.GetMessagesBySessionIDParams{
		Column1: uuidToPgUUID(sessionID),
		Column2: limit,
		Column3: offset,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get messages by session ID: %w", err)
	}

	return domainMessagesFromDB(dbMessages), nil
}

func (r *messageRepo) GetLastMessagesBySessionID(ctx context.Context, sessionID uuid.UUID, limit int32) ([]message.Message, error) {
	dbMessages, err := r.querier.GetLastMessagesBySessionID(ctx, sqlc.GetLastMessagesBySessionIDParams{
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

func (r *messageRepo) CountBySessionID(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	count, err := r.querier.CountMessages(ctx, uuidToPgUUID(sessionID))
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return 0, err
		}
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return count, nil
}
