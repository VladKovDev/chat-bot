package postgres

import (
	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository/postgres/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// domainConversationFromDB converts sqlc.Conversation to domain.Conversation
func domainConversationFromDB(dbConv sqlc.Conversation) conversation.Conversation {
	return conversation.Conversation{
		ID:      pgUUIDToUUID(dbConv.ID),
		Channel: conversation.Channel(dbConv.Channel),
		ChatID:  dbConv.ChatID,
		State:   conversation.State(dbConv.State),
	}
}

// domainConversationsFromDB converts a slice of sqlc.Conversation to domain.Conversation
func domainConversationsFromDB(dbConvs []sqlc.Conversation) []conversation.Conversation {
	convs := make([]conversation.Conversation, len(dbConvs))
	for i, dbConv := range dbConvs {
		convs[i] = domainConversationFromDB(dbConv)
	}
	return convs
}

// pgUUIDToUUID converts pgtype.UUID to google.uuid.UUID
func pgUUIDToUUID(id pgtype.UUID) uuid.UUID {
	if !id.Valid {
		return uuid.Nil
	}
	return uuid.UUID(id.Bytes)
}

// uuidToPgUUID converts google.uuid.UUID to pgtype.UUID
func uuidToPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: [16]byte(id),
		Valid: true,
	}
}