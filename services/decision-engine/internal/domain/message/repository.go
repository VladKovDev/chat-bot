package message

import (
	"context"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, msg Message) (Message, error)
	GetBySessionID(ctx context.Context, sessionID uuid.UUID, limit int32, offset int32) ([]Message, error)
	GetLastMessagesBySessionID(ctx context.Context, sessionID uuid.UUID, limit int32) ([]Message, error)
	CountBySessionID(ctx context.Context, sessionID uuid.UUID) (int64, error)
}
