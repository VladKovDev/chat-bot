-- name: CreateMessage :one
INSERT INTO messages ("session_id", "sender_type", "text", "intent")
VALUES ($1::UUID, $2::VARCHAR(16), $3::TEXT, $4::VARCHAR(50))
RETURNING *;

-- name: GetMessagesBySessionID :many
SELECT * FROM messages
WHERE "session_id" = $1::UUID
ORDER BY "created_at" ASC
LIMIT $2::INT OFFSET $3::INT;

-- name: GetLastMessagesBySessionID :many
SELECT * FROM messages
WHERE "session_id" = $1::UUID
ORDER BY "created_at" DESC
LIMIT $2::INT;

-- name: CountMessages :one
SELECT COUNT(*)::BIGINT FROM messages
WHERE "session_id" = $1::UUID;
