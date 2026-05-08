-- name: CreateSession :one
INSERT INTO "sessions" ("chat_id", "user_id", "state", "version", "status")
VALUES ($1::BIGINT, $2::UUID, $3::VARCHAR(50), 1, 'active')
RETURNING *;

-- name: GetSessionByChatID :one
SELECT * FROM "sessions"
WHERE "chat_id" = $1::BIGINT
ORDER BY "created_at" DESC
LIMIT 1;

-- name: GetSessionByID :one
SELECT * FROM "sessions"
WHERE "id" = $1::UUID;

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
