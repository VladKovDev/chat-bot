-- name: CreateConversation :one
INSERT INTO "conversations" ("chat_id", "state", "version")
VALUES ($1::BIGINT, $2::VARCHAR(50), 1)
RETURNING *;

-- name: GetConversationByChatID :one
SELECT * FROM "conversations"
WHERE "chat_id" = $1::BIGINT
ORDER BY "created_at" DESC
LIMIT 1;

-- name: GetConversationByID :one
SELECT * FROM "conversations"
WHERE "id" = $1::UUID;

-- name: UpdateConversationState :one
UPDATE "conversations"
SET "state" = $2::VARCHAR(50),
    "updated_at" = now()
WHERE "id" = $1::UUID
RETURNING *;

-- name: UpdateConversationWithVersion :one
UPDATE "conversations"
SET "state" = $2::VARCHAR(50),
    "version" = "version" + 1,
    "updated_at" = now()
WHERE "id" = $1::UUID
RETURNING *;

-- name: ListConversations :many
SELECT * FROM "conversations"
ORDER BY "updated_at" DESC
LIMIT $1::INT OFFSET $2::INT;

-- name: ListConversationsByState :many
SELECT * FROM "conversations"
WHERE "state" = $1::VARCHAR(50)
ORDER BY "updated_at" DESC
LIMIT $2::INT OFFSET $3::INT;

-- name: DeleteConversation :exec
DELETE FROM "conversations"
WHERE "id" = $1::UUID;


-- name: CountConversations :one
SELECT COUNT(*)::BIGINT FROM "conversations";