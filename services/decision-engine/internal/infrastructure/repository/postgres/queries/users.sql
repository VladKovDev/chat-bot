-- name: CreateUser :one
INSERT INTO users ("external_id")
VALUES ($1::TEXT)
ON CONFLICT ("external_id") DO NOTHING
RETURNING *;

-- name: GetUserByExternalID :one
SELECT * FROM users
WHERE "external_id" = $1::TEXT;

-- name: GetUserByID :one
SELECT * FROM users
WHERE "id" = $1::UUID;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY "created_at" DESC
LIMIT $1::INT OFFSET $2::INT;

-- name: CountUsers :one
SELECT COUNT(*)::BIGINT FROM users;
