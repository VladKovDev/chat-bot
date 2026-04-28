package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository/postgres/sqlc"
	"github.com/google/uuid"
)

type conversationRepo struct {
	pool    *Pool
	querier *sqlc.Queries
}

func NewConversationRepo(pool *Pool) conversation.Repository {
	return &conversationRepo{
		pool:    pool,
		querier: sqlc.New(pool.Pool),
	}
}

func (r *conversationRepo) Create(ctx context.Context, conv conversation.Conversation) (conversation.Conversation, error) {
	dbConv, err := r.querier.CreateConversation(ctx, sqlc.CreateConversationParams{
		Column1: conv.ChatID,
		Column2: string(conv.State),
	})
	if err != nil {
		return conversation.Conversation{}, fmt.Errorf("failed to create conversation: %w", err)
	}

	return domainConversationFromDB(dbConv), nil
}

func (r *conversationRepo) GetByID(ctx context.Context, id uuid.UUID) (conversation.Conversation, error) {
	dbConv, err := r.querier.GetConversationByID(ctx, uuidToPgUUID(id))
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return conversation.Conversation{}, err
		}
		return conversation.Conversation{}, conversation.ErrNotFound
	}

	return domainConversationFromDB(dbConv), nil
}

// GetByChatID retrieves a conversation by chat ID
func (r *conversationRepo) GetByChatID(
	ctx context.Context,
	chatID int64,
) (conversation.Conversation, error) {
	dbConv, err := r.querier.GetConversationByChatID(ctx, chatID)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return conversation.Conversation{}, err
		}
		return conversation.Conversation{}, conversation.ErrNotFound
	}

	return domainConversationFromDB(dbConv), nil
}

func (r *conversationRepo) UpdateState(
	ctx context.Context,
	id uuid.UUID,
	state state.State,
) (conversation.Conversation, error) {
	dbConv, err := r.querier.UpdateConversationState(ctx, sqlc.UpdateConversationStateParams{
		Column1: uuidToPgUUID(id),
		Column2: string(state),
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return conversation.Conversation{}, err
		}
		return conversation.Conversation{}, conversation.ErrNotFound
	}

	return domainConversationFromDB(dbConv), nil
}

func (r *conversationRepo) UpdateStateWithVersion(
	ctx context.Context,
	id uuid.UUID,
	state state.State,
) (conversation.Conversation, error) {
	dbConv, err := r.querier.UpdateConversationWithVersion(ctx, sqlc.UpdateConversationWithVersionParams{
		Column1: uuidToPgUUID(id),
		Column2: string(state),
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return conversation.Conversation{}, err
		}
		return conversation.Conversation{}, conversation.ErrNotFound
	}

	return domainConversationFromDB(dbConv), nil
}

func (r *conversationRepo) List(
	ctx context.Context,
	limit int32,
	offset int32,
) ([]conversation.Conversation, error) {
	dbConvs, err := r.querier.ListConversations(ctx, sqlc.ListConversationsParams{
		Column1: limit,
		Column2: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	return domainConversationsFromDB(dbConvs), nil
}

func (r *conversationRepo) ListByState(
	ctx context.Context,
	state state.State,
	limit int32,
	offset int32,
) ([]conversation.Conversation, error) {
	dbConvs, err := r.querier.ListConversationsByState(ctx, sqlc.ListConversationsByStateParams{
		Column1: string(state),
		Column2: limit,
		Column3: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations by state: %w", err)
	}

	return domainConversationsFromDB(dbConvs), nil
}

func (r *conversationRepo) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.querier.DeleteConversation(ctx, uuidToPgUUID(id))
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return conversation.ErrNotFound
	}
	return nil
}

func (r *conversationRepo) Count(ctx context.Context) (int64, error) {
	count, err := r.querier.CountConversations(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count conversations: %w", err)
	}
	return count, nil
}
