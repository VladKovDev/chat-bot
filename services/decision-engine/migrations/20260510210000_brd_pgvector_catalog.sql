-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS vector;

ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS idempotency_key TEXT,
    ADD COLUMN IF NOT EXISTS detected_intent VARCHAR(80),
    ADD COLUMN IF NOT EXISTS confidence DOUBLE PRECISION CHECK (confidence IS NULL OR (confidence >= 0 AND confidence <= 1)),
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS created_by VARCHAR(16) NOT NULL DEFAULT 'runtime'
        CHECK (created_by IN ('runtime', 'operator', 'system', 'seed'));

CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_session_idempotency_key
ON messages(session_id, idempotency_key)
WHERE idempotency_key IS NOT NULL AND idempotency_key <> '';

CREATE INDEX IF NOT EXISTS idx_messages_session_created_at
ON messages(session_id, created_at);

CREATE TABLE IF NOT EXISTS session_context (
    session_id UUID PRIMARY KEY REFERENCES "sessions"(id) ON DELETE CASCADE,
    active_topic VARCHAR(80) NOT NULL DEFAULT '',
    mode VARCHAR(32) NOT NULL DEFAULT 'standard'
        CHECK (mode IN ('standard', 'waiting_operator', 'operator_connected', 'closed')),
    last_intent VARCHAR(80) NOT NULL DEFAULT '',
    fallback_count INT NOT NULL DEFAULT 0 CHECK (fallback_count >= 0),
    operator_status VARCHAR(32) NOT NULL DEFAULT 'none'
        CHECK (operator_status IN ('none', 'waiting', 'connected', 'closed')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_session_context_mode_topic
ON session_context(mode, active_topic);

CREATE OR REPLACE FUNCTION sync_session_context_from_sessions()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO session_context (
        session_id,
        active_topic,
        mode,
        last_intent,
        fallback_count,
        operator_status,
        metadata,
        version,
        created_at,
        updated_at
    )
    VALUES (
        NEW.id,
        NEW.active_topic,
        NEW.mode,
        NEW.last_intent,
        NEW.fallback_count,
        NEW.operator_status,
        NEW.metadata,
        NEW.version,
        now(),
        now()
    )
    ON CONFLICT (session_id) DO UPDATE
    SET active_topic = EXCLUDED.active_topic,
        mode = EXCLUDED.mode,
        last_intent = EXCLUDED.last_intent,
        fallback_count = EXCLUDED.fallback_count,
        operator_status = EXCLUDED.operator_status,
        metadata = EXCLUDED.metadata,
        version = EXCLUDED.version,
        updated_at = now();

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_sync_session_context_from_sessions ON "sessions";
CREATE TRIGGER trg_sync_session_context_from_sessions
AFTER INSERT OR UPDATE OF active_topic, mode, last_intent, fallback_count, operator_status, metadata, version
ON "sessions"
FOR EACH ROW
EXECUTE FUNCTION sync_session_context_from_sessions();

CREATE TABLE IF NOT EXISTS message_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES "sessions"(id) ON DELETE CASCADE,
    message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    channel TEXT NOT NULL,
    external_event_id TEXT NOT NULL,
    event_type VARCHAR(40) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    received_at TIMESTAMP NOT NULL DEFAULT now(),
    processed_at TIMESTAMP,
    UNIQUE (channel, external_event_id)
);

CREATE INDEX IF NOT EXISTS idx_message_events_session_received_at
ON message_events(session_id, received_at);

CREATE TABLE IF NOT EXISTS intents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "key" VARCHAR(80) NOT NULL UNIQUE,
    category VARCHAR(80) NOT NULL,
    response_key VARCHAR(80) NOT NULL,
    requires_action BOOLEAN NOT NULL DEFAULT false,
    escalate_on_failure BOOLEAN NOT NULL DEFAULT false,
    fallback_policy VARCHAR(32) NOT NULL DEFAULT 'default'
        CHECK (fallback_policy IN ('default', 'operator', 'knowledge', 'none')),
    active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_intents_active_category
ON intents(active, category);

CREATE TABLE IF NOT EXISTS intent_examples (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    intent_id UUID NOT NULL REFERENCES intents(id) ON DELETE CASCADE,
    "text" TEXT NOT NULL,
    normalized_text TEXT NOT NULL DEFAULT '',
    embedding vector(384) NOT NULL,
    locale VARCHAR(8) NOT NULL DEFAULT 'ru',
    weight DOUBLE PRECISION NOT NULL DEFAULT 1 CHECK (weight > 0),
    active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_intent_examples_intent_id
ON intent_examples(intent_id);

CREATE INDEX IF NOT EXISTS idx_intent_examples_active_locale
ON intent_examples(active, locale);

CREATE UNIQUE INDEX IF NOT EXISTS idx_intent_examples_intent_locale_text
ON intent_examples(intent_id, locale, normalized_text);

CREATE INDEX IF NOT EXISTS idx_intent_examples_embedding_hnsw
ON intent_examples USING hnsw (embedding vector_cosine_ops);

CREATE TABLE IF NOT EXISTS knowledge_articles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "key" VARCHAR(120) NOT NULL UNIQUE,
    category VARCHAR(80) NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'seed',
    active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_knowledge_articles_active_category
ON knowledge_articles(active, category);

CREATE TABLE IF NOT EXISTS knowledge_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id UUID NOT NULL REFERENCES knowledge_articles(id) ON DELETE CASCADE,
    chunk_index INT NOT NULL CHECK (chunk_index >= 0),
    body TEXT NOT NULL,
    embedding vector(384) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE (article_id, chunk_index)
);

CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_article_id
ON knowledge_chunks(article_id);

CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_active
ON knowledge_chunks(active);

CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_embedding_hnsw
ON knowledge_chunks USING hnsw (embedding vector_cosine_ops);

CREATE TABLE IF NOT EXISTS quick_replies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    intent_id UUID REFERENCES intents(id) ON DELETE CASCADE,
    response_key VARCHAR(80) NOT NULL,
    label TEXT NOT NULL,
    action VARCHAR(40) NOT NULL DEFAULT 'send_message',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    sort_order INT NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    CHECK (intent_id IS NOT NULL OR response_key <> '')
);

CREATE INDEX IF NOT EXISTS idx_quick_replies_intent_order
ON quick_replies(intent_id, sort_order)
WHERE active = true;

CREATE INDEX IF NOT EXISTS idx_quick_replies_response_key_order
ON quick_replies(response_key, sort_order)
WHERE active = true;

ALTER TABLE decision_logs
    ADD COLUMN IF NOT EXISTS fallback_reason TEXT,
    ADD COLUMN IF NOT EXISTS threshold DOUBLE PRECISION CHECK (threshold IS NULL OR (threshold >= 0 AND threshold <= 1));

CREATE INDEX IF NOT EXISTS idx_decision_logs_session_created_at
ON decision_logs(session_id, created_at);

CREATE TABLE IF NOT EXISTS decision_candidates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_log_id UUID NOT NULL REFERENCES decision_logs(id) ON DELETE CASCADE,
    intent_id UUID REFERENCES intents(id) ON DELETE SET NULL,
    intent_key VARCHAR(80) NOT NULL,
    confidence DOUBLE PRECISION NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    rank INT NOT NULL CHECK (rank > 0),
    source VARCHAR(32) NOT NULL DEFAULT 'intent_example'
        CHECK (source IN ('intent_example', 'knowledge_chunk', 'exact_command', 'fallback')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE (decision_log_id, rank)
);

CREATE INDEX IF NOT EXISTS idx_decision_candidates_log_rank
ON decision_candidates(decision_log_id, rank);

ALTER TABLE actions_log
    ADD COLUMN IF NOT EXISTS message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS status VARCHAR(24) NOT NULL DEFAULT 'unknown'
        CHECK (status IN ('success', 'failure', 'skipped', 'unknown')),
    ADD COLUMN IF NOT EXISTS duration_ms INT CHECK (duration_ms IS NULL OR duration_ms >= 0),
    ADD COLUMN IF NOT EXISTS provider VARCHAR(80) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS redacted_payload JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_actions_log_message_id
ON actions_log(message_id)
WHERE message_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_actions_log_provider_status
ON actions_log(provider, status);

ALTER TABLE transitions_log
    ADD COLUMN IF NOT EXISTS message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS actor_type VARCHAR(16) NOT NULL DEFAULT 'system'
        CHECK (actor_type IN ('user', 'bot', 'operator', 'system'));

CREATE INDEX IF NOT EXISTS idx_transitions_log_message_id
ON transitions_log(message_id)
WHERE message_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS demo_accounts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL UNIQUE,
    identifiers TEXT[] NOT NULL DEFAULT '{}',
    email TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_demo_accounts_identifiers
ON demo_accounts USING gin (identifiers);

CREATE TABLE IF NOT EXISTS demo_bookings (
    id TEXT PRIMARY KEY,
    booking_number TEXT NOT NULL UNIQUE,
    identifiers TEXT[] NOT NULL DEFAULT '{}',
    service TEXT NOT NULL DEFAULT '',
    master TEXT NOT NULL DEFAULT '',
    booking_date DATE,
    booking_time TIME,
    status VARCHAR(32) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_demo_bookings_identifiers
ON demo_bookings USING gin (identifiers);

CREATE TABLE IF NOT EXISTS demo_workspace_bookings (
    id TEXT PRIMARY KEY,
    booking_number TEXT NOT NULL UNIQUE,
    identifiers TEXT[] NOT NULL DEFAULT '{}',
    workspace_type TEXT NOT NULL,
    booking_date DATE,
    booking_time TIME,
    duration_hours INT CHECK (duration_hours IS NULL OR duration_hours > 0),
    status VARCHAR(32) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_demo_workspace_bookings_identifiers
ON demo_workspace_bookings USING gin (identifiers);

CREATE TABLE IF NOT EXISTS demo_payments (
    id TEXT PRIMARY KEY,
    payment_id TEXT NOT NULL UNIQUE,
    identifiers TEXT[] NOT NULL DEFAULT '{}',
    amount_cents INT NOT NULL CHECK (amount_cents >= 0),
    paid_at TIMESTAMPTZ,
    status VARCHAR(32) NOT NULL,
    purpose TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_demo_payments_identifiers
ON demo_payments USING gin (identifiers);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS demo_payments CASCADE;
DROP TABLE IF EXISTS demo_workspace_bookings CASCADE;
DROP TABLE IF EXISTS demo_bookings CASCADE;
DROP TABLE IF EXISTS demo_accounts CASCADE;
DROP TABLE IF EXISTS decision_candidates CASCADE;
DROP TABLE IF EXISTS quick_replies CASCADE;
DROP TABLE IF EXISTS knowledge_chunks CASCADE;
DROP TABLE IF EXISTS knowledge_articles CASCADE;
DROP TABLE IF EXISTS intent_examples CASCADE;
DROP TABLE IF EXISTS intents CASCADE;
DROP INDEX IF EXISTS idx_intent_examples_intent_locale_text;
DROP TABLE IF EXISTS message_events CASCADE;
DROP TRIGGER IF EXISTS trg_sync_session_context_from_sessions ON "sessions";
DROP TABLE IF EXISTS session_context CASCADE;
DROP FUNCTION IF EXISTS sync_session_context_from_sessions();

DROP INDEX IF EXISTS idx_transitions_log_message_id;
DROP INDEX IF EXISTS idx_actions_log_provider_status;
DROP INDEX IF EXISTS idx_actions_log_message_id;
DROP INDEX IF EXISTS idx_decision_logs_session_created_at;
DROP INDEX IF EXISTS idx_messages_session_created_at;
DROP INDEX IF EXISTS idx_messages_session_idempotency_key;

ALTER TABLE transitions_log
    DROP COLUMN IF EXISTS actor_type,
    DROP COLUMN IF EXISTS message_id;

ALTER TABLE actions_log
    DROP COLUMN IF EXISTS redacted_payload,
    DROP COLUMN IF EXISTS provider,
    DROP COLUMN IF EXISTS duration_ms,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS message_id;

ALTER TABLE decision_logs
    DROP COLUMN IF EXISTS threshold,
    DROP COLUMN IF EXISTS fallback_reason;

ALTER TABLE messages
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS metadata,
    DROP COLUMN IF EXISTS confidence,
    DROP COLUMN IF EXISTS detected_intent,
    DROP COLUMN IF EXISTS idempotency_key;
-- +goose StatementEnd
