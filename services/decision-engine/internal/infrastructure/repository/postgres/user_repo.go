package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/user"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository/postgres/sqlc"
	"github.com/google/uuid"
)

type userRepo struct {
	pool    *Pool
	querier *sqlc.Queries
}

func NewUserRepo(pool *Pool) user.Repository {
	return &userRepo{
		pool:    pool,
		querier: sqlc.New(pool.Pool),
	}
}

func (r *userRepo) Create(ctx context.Context, u user.User) (user.User, error) {
	dbUser, err := r.querier.CreateUser(ctx, u.ExternalID)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return user.User{}, err
		}
		return user.User{}, fmt.Errorf("failed to create user: %w", err)
	}

	// ON CONFLICT returns empty, fetch the user if needed
	if !dbUser.ID.Valid {
		dbUser, err = r.querier.GetUserByExternalID(ctx, u.ExternalID)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return user.User{}, err
			}
			return user.User{}, fmt.Errorf("failed to get user by external ID: %w", err)
		}
	}

	return domainUserFromDB(dbUser), nil
}

func (r *userRepo) GetByExternalID(ctx context.Context, externalID string) (user.User, error) {
	dbUser, err := r.querier.GetUserByExternalID(ctx, externalID)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return user.User{}, err
		}
		return user.User{}, user.ErrNotFound
	}

	return domainUserFromDB(dbUser), nil
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (user.User, error) {
	dbUser, err := r.querier.GetUserByID(ctx, uuidToPgUUID(id))
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return user.User{}, err
		}
		return user.User{}, user.ErrNotFound
	}

	return domainUserFromDB(dbUser), nil
}

func (r *userRepo) List(ctx context.Context, limit int32, offset int32) ([]user.User, error) {
	dbUsers, err := r.querier.ListUsers(ctx, sqlc.ListUsersParams{
		Column1: limit,
		Column2: offset,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return domainUsersFromDB(dbUsers), nil
}

func (r *userRepo) Count(ctx context.Context) (int64, error) {
	count, err := r.querier.CountUsers(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return 0, err
		}
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}
