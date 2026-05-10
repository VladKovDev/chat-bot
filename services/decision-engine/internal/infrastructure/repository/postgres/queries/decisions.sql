-- name: LogDecision :one
INSERT INTO decision_logs (
    "session_id",
    "message_id",
    "intent",
    "state",
    "response_key",
    "confidence",
    "low_confidence",
    "candidates"
)
VALUES (
    sqlc.arg(session_id)::UUID,
    sqlc.arg(message_id)::UUID,
    sqlc.arg(intent)::VARCHAR(80),
    sqlc.arg(state)::VARCHAR(50),
    sqlc.arg(response_key)::VARCHAR(80),
    sqlc.narg(confidence)::DOUBLE PRECISION,
    sqlc.arg(low_confidence)::BOOLEAN,
    sqlc.arg(candidates)::JSONB
)
RETURNING *;
