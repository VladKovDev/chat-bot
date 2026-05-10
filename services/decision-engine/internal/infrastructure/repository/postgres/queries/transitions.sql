-- name: LogTransition :one
INSERT INTO transitions_log ("session_id", "from_state", "to_state", "event", "reason")
VALUES ($1::UUID, $2::VARCHAR(50), $3::VARCHAR(50), $4::VARCHAR(64), $5::TEXT)
RETURNING *;

-- name: GetTransitionsBySessionID :many
SELECT * FROM transitions_log
WHERE "session_id" = $1::UUID
ORDER BY "created_at" DESC
LIMIT $2::INT OFFSET $3::INT;

-- name: CountTransitions :one
SELECT COUNT(*)::BIGINT FROM transitions_log
WHERE "session_id" = $1::UUID;
