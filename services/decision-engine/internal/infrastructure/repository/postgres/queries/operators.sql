-- name: UpsertOperator :one
INSERT INTO operators (operator_id, fixture_id, display_name, status)
VALUES (
    sqlc.arg(operator_id)::TEXT,
    sqlc.arg(fixture_id)::TEXT,
    sqlc.arg(display_name)::TEXT,
    sqlc.arg(status)::VARCHAR(16)
)
ON CONFLICT (operator_id) DO UPDATE
SET fixture_id = EXCLUDED.fixture_id,
    display_name = EXCLUDED.display_name,
    status = EXCLUDED.status,
    updated_at = now()
RETURNING *;

-- name: CreateOperatorQueueItem :one
INSERT INTO operator_queue (
    id,
    session_id,
    user_id,
    reason,
    priority,
    context_snapshot
)
VALUES (
    sqlc.arg(id)::UUID,
    sqlc.arg(session_id)::UUID,
    sqlc.arg(user_id)::UUID,
    sqlc.arg(reason)::VARCHAR(40),
    sqlc.arg(priority)::INT,
    sqlc.arg(context_snapshot)::JSONB
)
RETURNING *;

-- name: GetOperatorQueueByID :one
SELECT * FROM operator_queue
WHERE id = $1::UUID;

-- name: GetOpenOperatorQueueBySession :one
SELECT * FROM operator_queue
WHERE session_id = $1::UUID
  AND status IN ('waiting', 'accepted')
ORDER BY created_at DESC
LIMIT 1;

-- name: ListOperatorQueueByStatus :many
SELECT * FROM operator_queue
WHERE status = $1::VARCHAR(16)
ORDER BY created_at ASC
LIMIT $2::INT OFFSET $3::INT;

-- name: AcceptOperatorQueueItem :one
UPDATE operator_queue
SET status = 'accepted',
    assigned_operator_id = sqlc.arg(operator_id)::TEXT,
    accepted_at = COALESCE(accepted_at, now()),
    updated_at = now()
WHERE id = sqlc.arg(id)::UUID
  AND status = 'waiting'
RETURNING *;

-- name: CloseOperatorQueueItem :one
UPDATE operator_queue
SET status = 'closed',
    closed_at = COALESCE(closed_at, now()),
    updated_at = now()
WHERE id = sqlc.arg(id)::UUID
  AND status IN ('waiting', 'accepted')
RETURNING *;

-- name: CreateOperatorAssignment :one
INSERT INTO operator_assignments (queue_id, operator_id, status)
VALUES (
    sqlc.arg(queue_id)::UUID,
    sqlc.arg(operator_id)::TEXT,
    'accepted'
)
RETURNING *;

-- name: CloseOperatorAssignment :exec
UPDATE operator_assignments
SET status = 'closed',
    released_at = COALESCE(released_at, now())
WHERE queue_id = $1::UUID
  AND status = 'accepted';

-- name: CreateOperatorEvent :one
INSERT INTO operator_events (
    queue_id,
    session_id,
    event_type,
    actor_type,
    actor_id,
    payload
)
VALUES (
    sqlc.arg(queue_id)::UUID,
    sqlc.arg(session_id)::UUID,
    sqlc.arg(event_type)::VARCHAR(24),
    sqlc.arg(actor_type)::VARCHAR(16),
    sqlc.arg(actor_id)::TEXT,
    sqlc.arg(payload)::JSONB
)
RETURNING *;
