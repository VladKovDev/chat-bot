-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS decision_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES "sessions"(id) ON DELETE CASCADE,
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    intent VARCHAR(80) NOT NULL,
    state VARCHAR(50) NOT NULL,
    response_key VARCHAR(80) NOT NULL,
    confidence DOUBLE PRECISION,
    low_confidence BOOLEAN NOT NULL DEFAULT false,
    candidates JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_decision_logs_session_id ON decision_logs(session_id);
CREATE INDEX idx_decision_logs_message_id ON decision_logs(message_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS decision_logs CASCADE;
-- +goose StatementEnd
