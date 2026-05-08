-- +goose Up
-- +goose StatementBegin
-- 1. Sessions
CREATE TABLE IF NOT EXISTS "sessions" (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id BIGINT NOT NULL,
    user_id UUID NOT NULL,
    "state" VARCHAR(50) NOT NULL,
    summary varchar(255) DEFAULT NULL,
    "version" INT NOT NULL DEFAULT 1,
    "status" VARCHAR(20) NOT NULL DEFAULT 'active' CHECK ("status" IN ('active', 'closed')),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_session_chat_id ON "sessions"(chat_id);

CREATE INDEX idx_session_user_id ON "sessions"(user_id);

CREATE INDEX idx_session_state ON "sessions"("state");

-- 2. Messages
CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES "sessions"(id) ON DELETE CASCADE,
    sender_type VARCHAR(16) NOT NULL CHECK (sender_type IN ('user', 'bot', 'operator')),
    "text" TEXT NOT NULL,
    intent VARCHAR(50),
    -- optional, filled after detection
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_messages_sender_type ON messages(sender_type);

CREATE INDEX idx_messages_session_id ON messages(session_id);

CREATE INDEX idx_messages_created_at ON messages(created_at);

-- 3. Users
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- 4. Transitions Log
CREATE TABLE IF NOT EXISTS transitions_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES "sessions"(id) ON DELETE CASCADE,
    from_state VARCHAR(50) NOT NULL,
    to_state VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_transitions_log_session_id ON transitions_log(session_id);

-- 5. Actions Log
CREATE TABLE IF NOT EXISTS actions_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES "sessions"(id) ON DELETE CASCADE,
    action_type VARCHAR(50) NOT NULL,
    request_payload JSONB,
    response_payload JSONB,
    error text,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_actions_log_session_id ON actions_log(session_id);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS messages CASCADE;

DROP TABLE IF EXISTS "sessions" CASCADE;

DROP TABLE IF EXISTS transitions_log CASCADE;

DROP TABLE IF EXISTS users CASCADE;

DROP TABLE IF EXISTS actions_log CASCADE;

-- +goose StatementEnd