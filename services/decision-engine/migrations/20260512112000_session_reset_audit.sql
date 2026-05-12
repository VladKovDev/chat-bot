-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS session_reset_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL,
    actor TEXT NOT NULL DEFAULT 'system',
    reason TEXT NOT NULL DEFAULT 'manual_reset',
    existed BOOLEAN NOT NULL DEFAULT false,
    deleted_counts JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_session_reset_audit_session_created_at
ON session_reset_audit(session_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS session_reset_audit CASCADE;
-- +goose StatementEnd
