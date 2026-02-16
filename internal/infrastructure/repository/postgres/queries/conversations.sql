-- name: CreateConversation :one
INSERT INTO "conversations" ("channel", "chat_id", "state", "version")
VALUES ($1::VARCHAR(50), $2::BIGINT, $3::VARCHAR(50), 1)
RETURNING *;

-- name: GetConversationByChannelAndChatID :one
SELECT * FROM "conversations"
WHERE "channel" = $1::VARCHAR(50)
  AND "chat_id" = $2::BIGINT
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

-- name: ListConversationsByChannel :many
SELECT * FROM "conversations"
WHERE "channel" = $1::VARCHAR(50)
ORDER BY "updated_at" DESC
LIMIT $2::INT OFFSET $3::INT;

-- name: ListConversationsByState :many
SELECT * FROM "conversations"
WHERE "state" = $1::VARCHAR(50)
ORDER BY "updated_at" DESC
LIMIT $2::INT OFFSET $3::INT;

-- name: DeleteConversation :exec
DELETE FROM "conversations"
WHERE "id" = $1::UUID;


-- name: CountConversationsByChannel :one
SELECT COUNT(*)::BIGINT FROM "conversations"
WHERE "channel" = $1::VARCHAR(50);