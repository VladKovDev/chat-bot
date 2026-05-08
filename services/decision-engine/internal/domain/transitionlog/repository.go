package transitionlog

import (
	"context"
	"github.com/google/uuid"
)

type Repository interface {
	Log(ctx context.Context, entry TransitionLog) (TransitionLog, error)
	GetBySessionID(ctx context.Context, sessionID uuid.UUID, limit int32, offset int32) ([]TransitionLog, error)
	CountBySessionID(ctx context.Context, sessionID uuid.UUID) (int64, error)
}
