package postgres

import (
	"encoding/json"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/message"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/internal/domain/transitionlog"
	"github.com/VladKovDev/chat-bot/internal/domain/user"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository/postgres/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// domainSessionFromDB converts sqlc.Session to domain.Session
func domainSessionFromDB(dbSession sqlc.Session) session.Session {
	metadata := make(map[string]interface{})
	if len(dbSession.Metadata) > 0 {
		_ = json.Unmarshal(dbSession.Metadata, &metadata)
	}

	return session.Session{
		ID:             pgUUIDToUUID(dbSession.ID),
		ChatID:         dbSession.ChatID,
		UserID:         pgUUIDToUUID(dbSession.UserID),
		Channel:        dbSession.Channel,
		ExternalUserID: dbSession.ExternalUserID,
		ClientID:       dbSession.ClientID,
		State:          state.State(dbSession.State),
		Mode:           session.Mode(dbSession.Mode),
		ActiveTopic:    dbSession.ActiveTopic,
		LastIntent:     dbSession.LastIntent,
		FallbackCount:  int(dbSession.FallbackCount),
		OperatorStatus: session.OperatorStatus(dbSession.OperatorStatus),
		Summary:        dbSession.Summary,
		Version:        int(dbSession.Version),
		Status:         session.Status(dbSession.Status),
		Metadata:       metadata,
	}
}

// domainSessionsFromDB converts a slice of sqlc.Session to domain.Session
func domainSessionsFromDB(dbSessions []sqlc.Session) []session.Session {
	sessions := make([]session.Session, len(dbSessions))
	for i, dbSession := range dbSessions {
		sessions[i] = domainSessionFromDB(dbSession)
	}
	return sessions
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

// Backward compatibility aliases
var domainConversationFromDB = domainSessionFromDB
var domainConversationsFromDB = domainSessionsFromDB

// domainMessageFromDB converts sqlc.Message to domain.Message
func domainMessageFromDB(dbMsg sqlc.Message) message.Message {
	var intentPtr *string
	if dbMsg.Intent != nil {
		intentPtr = dbMsg.Intent
	}
	return message.Message{
		ID:         pgUUIDToUUID(dbMsg.ID),
		SessionID:  pgUUIDToUUID(dbMsg.SessionID),
		SenderType: message.SenderType(dbMsg.SenderType),
		Text:       dbMsg.Text,
		Intent:     intentPtr,
		CreatedAt:  dbMsg.CreatedAt.Time,
	}
}

// domainMessagesFromDB converts a slice of sqlc.Message to domain.Message
func domainMessagesFromDB(dbMessages []sqlc.Message) []message.Message {
	messages := make([]message.Message, len(dbMessages))
	for i, dbMsg := range dbMessages {
		messages[i] = domainMessageFromDB(dbMsg)
	}
	return messages
}

// domainUserFromDB converts sqlc.User to domain.User
func domainUserFromDB(dbUser sqlc.User) user.User {
	return user.User{
		ID:         pgUUIDToUUID(dbUser.ID),
		ExternalID: dbUser.ExternalID,
		CreatedAt:  dbUser.CreatedAt.Time,
		UpdatedAt:  dbUser.UpdatedAt.Time,
	}
}

// domainUsersFromDB converts a slice of sqlc.User to domain.User
func domainUsersFromDB(dbUsers []sqlc.User) []user.User {
	users := make([]user.User, len(dbUsers))
	for i, dbUser := range dbUsers {
		users[i] = domainUserFromDB(dbUser)
	}
	return users
}

// domainTransitionLogFromDB converts sqlc.TransitionsLog to domain.TransitionLog
func domainTransitionLogFromDB(dbLog sqlc.TransitionsLog) transitionlog.TransitionLog {
	return transitionlog.TransitionLog{
		ID:        pgUUIDToUUID(dbLog.ID),
		SessionID: pgUUIDToUUID(dbLog.SessionID),
		FromMode:  session.Mode(dbLog.FromState),
		ToMode:    session.Mode(dbLog.ToState),
		Event:     session.Event(dbLog.Event),
		Reason:    dbLog.Reason,
		CreatedAt: dbLog.CreatedAt.Time,
	}
}

// domainTransitionLogsFromDB converts a slice of sqlc.TransitionsLog to domain.TransitionLog
func domainTransitionLogsFromDB(dbLogs []sqlc.TransitionsLog) []transitionlog.TransitionLog {
	logs := make([]transitionlog.TransitionLog, len(dbLogs))
	for i, dbLog := range dbLogs {
		logs[i] = domainTransitionLogFromDB(dbLog)
	}
	return logs
}

// domainActionLogFromDB converts sqlc.ActionsLog to domain.Action.Log
func domainActionLogFromDB(dbLog sqlc.ActionsLog) action.Log {
	var requestPayload, responsePayload map[string]interface{}

	if dbLog.RequestPayload != nil {
		json.Unmarshal(dbLog.RequestPayload, &requestPayload)
	}

	if dbLog.ResponsePayload != nil {
		json.Unmarshal(dbLog.ResponsePayload, &responsePayload)
	}

	return action.Log{
		ID:              pgUUIDToUUID(dbLog.ID),
		SessionID:       pgUUIDToUUID(dbLog.SessionID),
		ActionType:      dbLog.ActionType,
		RequestPayload:  requestPayload,
		ResponsePayload: responsePayload,
		Error:           dbLog.Error,
		CreatedAt:       dbLog.CreatedAt.Time,
	}
}

// domainActionLogsFromDB converts a slice of sqlc.ActionsLog to domain.Action.Log
func domainActionLogsFromDB(dbLogs []sqlc.ActionsLog) []action.Log {
	logs := make([]action.Log, len(dbLogs))
	for i, dbLog := range dbLogs {
		logs[i] = domainActionLogFromDB(dbLog)
	}
	return logs
}
