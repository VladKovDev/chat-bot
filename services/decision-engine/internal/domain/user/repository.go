package user

import (
	"context"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, u User) (User, error)
	GetByExternalID(ctx context.Context, externalID string) (User, error)
	GetByID(ctx context.Context, id uuid.UUID) (User, error)
	List(ctx context.Context, limit int32, offset int32) ([]User, error)
	Count(ctx context.Context) (int64, error)
}
