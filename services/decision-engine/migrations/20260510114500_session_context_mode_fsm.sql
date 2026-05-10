-- +goose Up
-- +goose StatementBegin
ALTER TABLE "sessions"
    ADD COLUMN IF NOT EXISTS mode VARCHAR(32) NOT NULL DEFAULT 'standard'
        CHECK (mode IN ('standard', 'waiting_operator', 'operator_connected', 'closed')),
    ADD COLUMN IF NOT EXISTS last_intent VARCHAR(80) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS fallback_count INT NOT NULL DEFAULT 0 CHECK (fallback_count >= 0),
    ADD COLUMN IF NOT EXISTS operator_status VARCHAR(32) NOT NULL DEFAULT 'none'
        CHECK (operator_status IN ('none', 'waiting', 'connected', 'closed')),
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_session_mode ON "sessions"(mode);
CREATE INDEX IF NOT EXISTS idx_session_active_topic_mode ON "sessions"(active_topic, mode);

ALTER TABLE transitions_log
    ADD COLUMN IF NOT EXISTS event VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS reason TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE transitions_log
    DROP COLUMN IF EXISTS reason,
    DROP COLUMN IF EXISTS event;

DROP INDEX IF EXISTS idx_session_active_topic_mode;
DROP INDEX IF EXISTS idx_session_mode;

ALTER TABLE "sessions"
    DROP COLUMN IF EXISTS metadata,
    DROP COLUMN IF EXISTS operator_status,
    DROP COLUMN IF EXISTS fallback_count,
    DROP COLUMN IF EXISTS last_intent,
    DROP COLUMN IF EXISTS mode;
-- +goose StatementEnd
