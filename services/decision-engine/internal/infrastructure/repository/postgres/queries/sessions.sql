-- name: CreateSession :one
INSERT INTO "sessions" (
    "chat_id",
    "user_id",
    "channel",
    "external_user_id",
    "client_id",
    "state",
    "mode",
    "active_topic",
    "last_intent",
    "fallback_count",
    "operator_status",
    "metadata",
    "version",
    "status"
)
VALUES (
    sqlc.arg(chat_id)::BIGINT,
    sqlc.arg(user_id)::UUID,
    sqlc.arg(channel)::TEXT,
    sqlc.arg(external_user_id)::TEXT,
    sqlc.arg(client_id)::TEXT,
    sqlc.arg(state)::VARCHAR(50),
    sqlc.arg(mode)::VARCHAR(32),
    sqlc.arg(active_topic)::VARCHAR(50),
    sqlc.arg(last_intent)::VARCHAR(80),
    sqlc.arg(fallback_count)::INT,
    sqlc.arg(operator_status)::VARCHAR(32),
    sqlc.arg(metadata)::JSONB,
    1,
    'active'
)
RETURNING *;

-- name: GetSessionByChatID :one
SELECT * FROM "sessions"
WHERE "chat_id" = $1::BIGINT
ORDER BY "created_at" DESC
LIMIT 1;

-- name: GetActiveSessionByIdentity :one
SELECT * FROM "sessions"
WHERE "channel" = sqlc.arg(channel)::TEXT
  AND "status" = 'active'
  AND (
      (sqlc.arg(external_user_id)::TEXT <> '' AND "external_user_id" = sqlc.arg(external_user_id)::TEXT)
      OR (sqlc.arg(external_user_id)::TEXT = '' AND "client_id" = sqlc.arg(client_id)::TEXT)
  )
ORDER BY "created_at" DESC
LIMIT 1;

-- name: GetSessionByID :one
SELECT * FROM "sessions"
WHERE "id" = $1::UUID;

-- name: UpdateSession :one
UPDATE "sessions"
SET "state" = sqlc.arg(state)::VARCHAR(50),
    "mode" = sqlc.arg(mode)::VARCHAR(32),
    "active_topic" = sqlc.arg(active_topic)::VARCHAR(50),
    "last_intent" = sqlc.arg(last_intent)::VARCHAR(80),
    "fallback_count" = sqlc.arg(fallback_count)::INT,
    "operator_status" = sqlc.arg(operator_status)::VARCHAR(32),
    "metadata" = sqlc.arg(metadata)::JSONB,
    "status" = sqlc.arg(status)::VARCHAR(20),
    "version" = "version" + 1,
    "updated_at" = now()
WHERE "id" = sqlc.arg(id)::UUID
RETURNING *;

-- name: UpdateSessionState :one
UPDATE "sessions"
SET "state" = $2::VARCHAR(50),
    "updated_at" = now()
WHERE "id" = $1::UUID
RETURNING *;

-- name: UpdateSessionWithVersion :one
UPDATE "sessions"
SET "state" = $2::VARCHAR(50),
    "version" = "version" + 1,
    "updated_at" = now()
WHERE "id" = $1::UUID
RETURNING *;

-- name: UpdateSessionStatus :one
UPDATE "sessions"
SET "status" = $2::VARCHAR(20),
    "updated_at" = now()
WHERE "id" = $1::UUID
RETURNING *;

-- name: UpdateSessionSummary :one
UPDATE "sessions"
SET "summary" = $2::VARCHAR(255),
    "updated_at" = now()
WHERE "id" = $1::UUID
RETURNING *;

-- name: ListSessions :many
SELECT * FROM "sessions"
ORDER BY "updated_at" DESC
LIMIT $1::INT OFFSET $2::INT;

-- name: ListSessionsByState :many
SELECT * FROM "sessions"
WHERE "state" = $1::VARCHAR(50)
ORDER BY "updated_at" DESC
LIMIT $2::INT OFFSET $3::INT;

-- name: ListSessionsByStatus :many
SELECT * FROM "sessions"
WHERE "status" = $1::VARCHAR(20)
ORDER BY "updated_at" DESC
LIMIT $2::INT OFFSET $3::INT;

-- name: ListSessionsByUser :many
SELECT * FROM "sessions"
WHERE "user_id" = $1::UUID
ORDER BY "updated_at" DESC
LIMIT $2::INT OFFSET $3::INT;

-- name: DeleteSession :exec
DELETE FROM "sessions"
WHERE "id" = $1::UUID;

-- name: CountSessions :one
SELECT COUNT(*)::BIGINT FROM "sessions";

-- name: CloseSession :one
UPDATE "sessions"
SET "status" = 'closed',
    "updated_at" = now()
WHERE "id" = $1::UUID
RETURNING *;
