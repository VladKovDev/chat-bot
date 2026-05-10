-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS operators (
    operator_id TEXT PRIMARY KEY,
    fixture_id TEXT UNIQUE,
    display_name TEXT NOT NULL,
    status VARCHAR(16) NOT NULL CHECK (status IN ('available', 'busy', 'offline')),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS operator_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES "sessions"(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status VARCHAR(16) NOT NULL DEFAULT 'waiting'
        CHECK (status IN ('waiting', 'accepted', 'closed')),
    reason VARCHAR(40) NOT NULL
        CHECK (reason IN ('manual_request', 'low_confidence_repeated', 'complaint', 'business_error')),
    priority INT NOT NULL DEFAULT 0 CHECK (priority >= 0),
    assigned_operator_id TEXT REFERENCES operators(operator_id) ON DELETE RESTRICT,
    context_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    accepted_at TIMESTAMP,
    closed_at TIMESTAMP
);

CREATE UNIQUE INDEX idx_operator_queue_open_session
ON operator_queue(session_id)
WHERE status IN ('waiting', 'accepted');

CREATE INDEX idx_operator_queue_status_created_at
ON operator_queue(status, created_at);

CREATE INDEX idx_operator_queue_assigned_operator_status
ON operator_queue(assigned_operator_id, status);

CREATE TABLE IF NOT EXISTS operator_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    queue_id UUID NOT NULL REFERENCES operator_queue(id) ON DELETE CASCADE,
    operator_id TEXT NOT NULL REFERENCES operators(operator_id) ON DELETE RESTRICT,
    status VARCHAR(16) NOT NULL DEFAULT 'accepted'
        CHECK (status IN ('accepted', 'closed')),
    assigned_at TIMESTAMP NOT NULL DEFAULT now(),
    released_at TIMESTAMP
);

CREATE UNIQUE INDEX idx_operator_assignments_open_queue
ON operator_assignments(queue_id)
WHERE status = 'accepted';

CREATE INDEX idx_operator_assignments_operator_status
ON operator_assignments(operator_id, status);

CREATE TABLE IF NOT EXISTS operator_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    queue_id UUID NOT NULL REFERENCES operator_queue(id) ON DELETE CASCADE,
    session_id UUID NOT NULL REFERENCES "sessions"(id) ON DELETE CASCADE,
    event_type VARCHAR(24) NOT NULL CHECK (event_type IN ('queued', 'accepted', 'closed')),
    actor_type VARCHAR(16) NOT NULL CHECK (actor_type IN ('user', 'operator', 'system')),
    actor_id TEXT,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_operator_events_queue_created_at
ON operator_events(queue_id, created_at);

CREATE INDEX idx_operator_events_session_created_at
ON operator_events(session_id, created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS operator_events CASCADE;
DROP TABLE IF EXISTS operator_assignments CASCADE;
DROP TABLE IF EXISTS operator_queue CASCADE;
DROP TABLE IF EXISTS operators CASCADE;
-- +goose StatementEnd
