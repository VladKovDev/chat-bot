-- name: LogAction :one
INSERT INTO actions_log ("session_id", "action_type", "request_payload", "response_payload", "error")
VALUES ($1::UUID, $2::VARCHAR(50), $3::JSONB, $4::JSONB, $5::TEXT)
RETURNING *;

-- name: GetActionsBySessionID :many
SELECT * FROM actions_log
WHERE "session_id" = $1::UUID
ORDER BY "created_at" DESC
LIMIT $2::INT OFFSET $3::INT;

-- name: GetActionsByType :many
SELECT * FROM actions_log
WHERE "action_type" = $1::VARCHAR(50)
ORDER BY "created_at" DESC
LIMIT $2::INT OFFSET $3::INT;

-- name: CountActions :one
SELECT COUNT(*)::BIGINT FROM actions_log
WHERE "session_id" = $1::UUID;
