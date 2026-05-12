package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	domaindialogreset "github.com/VladKovDev/chat-bot/internal/domain/dialogreset"
	"github.com/jackc/pgx/v5/pgtype"
)

type dialogResetRepo struct {
	pool *Pool
}

func NewDialogResetRepo(pool *Pool) domaindialogreset.Repository {
	return &dialogResetRepo{pool: pool}
}

func (r *dialogResetRepo) ResetSession(
	ctx context.Context,
	req domaindialogreset.Request,
) (domaindialogreset.Summary, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domaindialogreset.Summary{}, fmt.Errorf("failed to begin dialog reset transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var existed bool
	sessionID := uuidToPgUUID(req.SessionID)
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM "sessions" WHERE id = $1::UUID)`, sessionID).Scan(&existed); err != nil {
		return domaindialogreset.Summary{}, fmt.Errorf("failed to check reset session existence: %w", err)
	}

	deleted := emptyResetCounts()
	if existed {
		var messages, decisionLogs, decisionCandidates, actionLogs, transitionLogs int64
		var sessionContext, messageEvents, operatorQueue, operatorAssignments, operatorEvents int64
		if err := tx.QueryRow(ctx, resetCountsSQL, sessionID).Scan(
			&messages,
			&decisionLogs,
			&decisionCandidates,
			&actionLogs,
			&transitionLogs,
			&sessionContext,
			&messageEvents,
			&operatorQueue,
			&operatorAssignments,
			&operatorEvents,
		); err != nil {
			return domaindialogreset.Summary{}, fmt.Errorf("failed to count reset session tails: %w", err)
		}
		deleted["messages"] = messages
		deleted["decision_logs"] = decisionLogs
		deleted["decision_candidates"] = decisionCandidates
		deleted["actions_log"] = actionLogs
		deleted["transitions_log"] = transitionLogs
		deleted["session_context"] = sessionContext
		deleted["message_events"] = messageEvents
		deleted["operator_queue"] = operatorQueue
		deleted["operator_assignments"] = operatorAssignments
		deleted["operator_events"] = operatorEvents
	}

	deletedJSON, err := json.Marshal(deleted)
	if err != nil {
		return domaindialogreset.Summary{}, fmt.Errorf("failed to marshal reset deleted counts: %w", err)
	}

	var auditID pgtype.UUID
	var createdAt pgtype.Timestamp
	if err := tx.QueryRow(ctx, `
		INSERT INTO session_reset_audit (session_id, actor, reason, existed, deleted_counts)
		VALUES ($1::UUID, $2::TEXT, $3::TEXT, $4::BOOLEAN, $5::JSONB)
		RETURNING id, created_at
	`, sessionID, req.Actor, req.Reason, existed, deletedJSON).Scan(&auditID, &createdAt); err != nil {
		return domaindialogreset.Summary{}, fmt.Errorf("failed to write reset audit: %w", err)
	}

	if existed {
		if _, err := tx.Exec(ctx, `DELETE FROM "sessions" WHERE id = $1::UUID`, sessionID); err != nil {
			return domaindialogreset.Summary{}, fmt.Errorf("failed to delete reset session: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domaindialogreset.Summary{}, fmt.Errorf("failed to commit dialog reset transaction: %w", err)
	}

	return domaindialogreset.Summary{
		SessionID: req.SessionID,
		Existed:   existed,
		Deleted:   deleted,
		AuditID:   pgUUIDToUUID(auditID),
		CreatedAt: createdAt.Time,
	}, nil
}

func emptyResetCounts() map[string]int64 {
	return map[string]int64{
		"messages":             0,
		"decision_logs":        0,
		"decision_candidates":  0,
		"actions_log":          0,
		"transitions_log":      0,
		"session_context":      0,
		"message_events":       0,
		"operator_queue":       0,
		"operator_assignments": 0,
		"operator_events":      0,
	}
}

const resetCountsSQL = `
SELECT
    (SELECT COUNT(*)::BIGINT FROM messages WHERE session_id = $1::UUID),
    (SELECT COUNT(*)::BIGINT FROM decision_logs WHERE session_id = $1::UUID),
    (
        SELECT COUNT(*)::BIGINT
        FROM decision_candidates dc
        JOIN decision_logs dl ON dl.id = dc.decision_log_id
        WHERE dl.session_id = $1::UUID
    ),
    (SELECT COUNT(*)::BIGINT FROM actions_log WHERE session_id = $1::UUID),
    (SELECT COUNT(*)::BIGINT FROM transitions_log WHERE session_id = $1::UUID),
    (SELECT COUNT(*)::BIGINT FROM session_context WHERE session_id = $1::UUID),
    (SELECT COUNT(*)::BIGINT FROM message_events WHERE session_id = $1::UUID),
    (SELECT COUNT(*)::BIGINT FROM operator_queue WHERE session_id = $1::UUID),
    (
        SELECT COUNT(*)::BIGINT
        FROM operator_assignments oa
        JOIN operator_queue oq ON oq.id = oa.queue_id
        WHERE oq.session_id = $1::UUID
    ),
    (SELECT COUNT(*)::BIGINT FROM operator_events WHERE session_id = $1::UUID)
`

var _ domaindialogreset.Repository = (*dialogResetRepo)(nil)
