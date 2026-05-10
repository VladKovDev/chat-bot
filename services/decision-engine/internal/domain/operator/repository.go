package operator

import (
	"context"

	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/google/uuid"
)

type Repository interface {
	UpsertOperator(ctx context.Context, account Account) (Account, error)
	Queue(ctx context.Context, req QueueRequest, sessionUpdate session.Session, transition *session.ModeTransition) (QueueItem, error)
	Accept(ctx context.Context, req AcceptRequest, sessionUpdate session.Session, transition *session.ModeTransition) (QueueItem, error)
	Close(ctx context.Context, req CloseRequest, sessionUpdate session.Session, transition *session.ModeTransition) (QueueItem, error)
	GetByID(ctx context.Context, id uuid.UUID) (QueueItem, error)
	GetOpenBySession(ctx context.Context, sessionID uuid.UUID) (QueueItem, error)
	ListByStatus(ctx context.Context, status QueueStatus, limit int32, offset int32) ([]QueueItem, error)
}
