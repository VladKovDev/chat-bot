-- +goose Up
-- +goose StatementBegin
-- 1. Conversations
CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel VARCHAR(50) NOT NULL,
    chat_id BIGINT NOT NULL,
    "state" VARCHAR(50) NOT NULL,
    "version" INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_conversations_chat_id ON conversations(channel, chat_id);

-- 2. Messages
CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_type VARCHAR(20) NOT NULL CHECK (sender_type IN ('user', 'bot', 'operator', 'system')),
    payload JSONB NOT NULL,
    intent VARCHAR(50),
    -- optional, filled after detection
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);

CREATE INDEX idx_messages_created_at ON messages(created_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS messages CASCADE;

DROP TABLE IF EXISTS conversations CASCADE;
-- +goose StatementEnd
