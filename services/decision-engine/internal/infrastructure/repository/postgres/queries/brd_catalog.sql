-- name: UpsertSessionContext :one
INSERT INTO session_context (
    session_id,
    active_topic,
    mode,
    last_intent,
    fallback_count,
    operator_status,
    metadata,
    version
)
VALUES (
    sqlc.arg(session_id)::UUID,
    sqlc.arg(active_topic)::VARCHAR(80),
    sqlc.arg(mode)::VARCHAR(32),
    sqlc.arg(last_intent)::VARCHAR(80),
    sqlc.arg(fallback_count)::INT,
    sqlc.arg(operator_status)::VARCHAR(32),
    sqlc.arg(metadata)::JSONB,
    1
)
ON CONFLICT (session_id) DO UPDATE
SET active_topic = EXCLUDED.active_topic,
    mode = EXCLUDED.mode,
    last_intent = EXCLUDED.last_intent,
    fallback_count = EXCLUDED.fallback_count,
    operator_status = EXCLUDED.operator_status,
    metadata = EXCLUDED.metadata,
    version = session_context.version + 1,
    updated_at = now()
RETURNING *;

-- name: GetSessionContext :one
SELECT * FROM session_context
WHERE session_id = $1::UUID;

-- name: CreateMessageEvent :one
INSERT INTO message_events (
    session_id,
    message_id,
    channel,
    external_event_id,
    event_type,
    payload,
    processed_at
)
VALUES (
    sqlc.arg(session_id)::UUID,
    sqlc.narg(message_id)::UUID,
    sqlc.arg(channel)::TEXT,
    sqlc.arg(external_event_id)::TEXT,
    sqlc.arg(event_type)::VARCHAR(40),
    sqlc.arg(payload)::JSONB,
    sqlc.narg(processed_at)::TIMESTAMP
)
RETURNING *;

-- name: GetMessageEventByExternalID :one
SELECT * FROM message_events
WHERE channel = sqlc.arg(channel)::TEXT
  AND external_event_id = sqlc.arg(external_event_id)::TEXT;

-- name: UpsertIntent :one
INSERT INTO intents (
    key,
    category,
    response_key,
    requires_action,
    escalate_on_failure,
    fallback_policy,
    active,
    metadata
)
VALUES (
    sqlc.arg(key)::VARCHAR(80),
    sqlc.arg(category)::VARCHAR(80),
    sqlc.arg(response_key)::VARCHAR(80),
    sqlc.arg(requires_action)::BOOLEAN,
    sqlc.arg(escalate_on_failure)::BOOLEAN,
    sqlc.arg(fallback_policy)::VARCHAR(32),
    sqlc.arg(active)::BOOLEAN,
    sqlc.arg(metadata)::JSONB
)
ON CONFLICT (key) DO UPDATE
SET category = EXCLUDED.category,
    response_key = EXCLUDED.response_key,
    requires_action = EXCLUDED.requires_action,
    escalate_on_failure = EXCLUDED.escalate_on_failure,
    fallback_policy = EXCLUDED.fallback_policy,
    active = EXCLUDED.active,
    metadata = EXCLUDED.metadata,
    updated_at = now()
RETURNING *;

-- name: GetIntentByKey :one
SELECT * FROM intents
WHERE key = $1::VARCHAR(80);

-- name: ListActiveIntents :many
SELECT * FROM intents
WHERE active = true
ORDER BY category, key;

-- name: UpsertIntentExample :one
INSERT INTO intent_examples (
    intent_id,
    text,
    normalized_text,
    embedding,
    locale,
    weight,
    active,
    metadata
)
VALUES (
    sqlc.arg(intent_id)::UUID,
    sqlc.arg(text)::TEXT,
    sqlc.arg(normalized_text)::TEXT,
    (sqlc.arg(embedding)::TEXT)::vector,
    sqlc.arg(locale)::VARCHAR(8),
    sqlc.arg(weight)::DOUBLE PRECISION,
    sqlc.arg(active)::BOOLEAN,
    sqlc.arg(metadata)::JSONB
)
ON CONFLICT (intent_id, locale, normalized_text) DO UPDATE
SET text = EXCLUDED.text,
    embedding = EXCLUDED.embedding,
    weight = EXCLUDED.weight,
    active = EXCLUDED.active,
    metadata = EXCLUDED.metadata,
    updated_at = now()
RETURNING id, intent_id, text, normalized_text, locale, weight, active, metadata, created_at, updated_at;

-- name: SearchIntentExamples :many
SELECT
    intent_examples.id,
    intent_examples.intent_id,
    intents.key AS intent_key,
    intents.category,
    intents.response_key,
    intent_examples.text,
    intent_examples.normalized_text,
    intent_examples.locale,
    intent_examples.weight,
    (1 - (intent_examples.embedding <=> (sqlc.arg(embedding)::TEXT)::vector))::DOUBLE PRECISION AS confidence
FROM intent_examples
JOIN intents ON intents.id = intent_examples.intent_id
WHERE intent_examples.active = true
  AND intents.active = true
  AND intent_examples.locale = sqlc.arg(locale)::VARCHAR(8)
ORDER BY intent_examples.embedding <=> (sqlc.arg(embedding)::TEXT)::vector
LIMIT sqlc.arg(result_limit)::INT;

-- name: UpsertKnowledgeArticle :one
INSERT INTO knowledge_articles (
    key,
    category,
    title,
    body,
    source,
    active,
    metadata
)
VALUES (
    sqlc.arg(key)::VARCHAR(120),
    sqlc.arg(category)::VARCHAR(80),
    sqlc.arg(title)::TEXT,
    sqlc.arg(body)::TEXT,
    sqlc.arg(source)::TEXT,
    sqlc.arg(active)::BOOLEAN,
    sqlc.arg(metadata)::JSONB
)
ON CONFLICT (key) DO UPDATE
SET category = EXCLUDED.category,
    title = EXCLUDED.title,
    body = EXCLUDED.body,
    source = EXCLUDED.source,
    active = EXCLUDED.active,
    metadata = EXCLUDED.metadata,
    updated_at = now()
RETURNING *;

-- name: UpsertKnowledgeChunk :one
INSERT INTO knowledge_chunks (
    article_id,
    chunk_index,
    body,
    embedding,
    active,
    metadata
)
VALUES (
    sqlc.arg(article_id)::UUID,
    sqlc.arg(chunk_index)::INT,
    sqlc.arg(body)::TEXT,
    (sqlc.arg(embedding)::TEXT)::vector,
    sqlc.arg(active)::BOOLEAN,
    sqlc.arg(metadata)::JSONB
)
ON CONFLICT (article_id, chunk_index) DO UPDATE
SET body = EXCLUDED.body,
    embedding = EXCLUDED.embedding,
    active = EXCLUDED.active,
    metadata = EXCLUDED.metadata,
    updated_at = now()
RETURNING id, article_id, chunk_index, body, active, metadata, created_at, updated_at;

-- name: SearchKnowledgeChunks :many
SELECT
    knowledge_chunks.id,
    knowledge_chunks.article_id,
    knowledge_articles.key AS article_key,
    knowledge_articles.category,
    knowledge_articles.title,
    knowledge_chunks.chunk_index,
    knowledge_chunks.body,
    (1 - (knowledge_chunks.embedding <=> (sqlc.arg(embedding)::TEXT)::vector))::DOUBLE PRECISION AS confidence
FROM knowledge_chunks
JOIN knowledge_articles ON knowledge_articles.id = knowledge_chunks.article_id
WHERE knowledge_chunks.active = true
  AND knowledge_articles.active = true
ORDER BY knowledge_chunks.embedding <=> (sqlc.arg(embedding)::TEXT)::vector
LIMIT sqlc.arg(result_limit)::INT;

-- name: UpsertQuickReply :one
INSERT INTO quick_replies (
    intent_id,
    response_key,
    label,
    action,
    payload,
    sort_order,
    active
)
VALUES (
    sqlc.narg(intent_id)::UUID,
    sqlc.arg(response_key)::VARCHAR(80),
    sqlc.arg(label)::TEXT,
    sqlc.arg(action)::VARCHAR(40),
    sqlc.arg(payload)::JSONB,
    sqlc.arg(sort_order)::INT,
    sqlc.arg(active)::BOOLEAN
)
RETURNING *;

-- name: ListQuickRepliesByIntent :many
SELECT * FROM quick_replies
WHERE intent_id = $1::UUID
  AND active = true
ORDER BY sort_order, label;

-- name: ListQuickRepliesByResponseKey :many
SELECT * FROM quick_replies
WHERE response_key = $1::VARCHAR(80)
  AND active = true
ORDER BY sort_order, label;

-- name: LogDecisionCandidate :one
INSERT INTO decision_candidates (
    decision_log_id,
    intent_id,
    intent_key,
    confidence,
    rank,
    source,
    metadata
)
VALUES (
    sqlc.arg(decision_log_id)::UUID,
    sqlc.narg(intent_id)::UUID,
    sqlc.arg(intent_key)::VARCHAR(80),
    sqlc.arg(confidence)::DOUBLE PRECISION,
    sqlc.arg(rank)::INT,
    sqlc.arg(source)::VARCHAR(32),
    sqlc.arg(metadata)::JSONB
)
RETURNING *;

-- name: ListDecisionCandidates :many
SELECT * FROM decision_candidates
WHERE decision_log_id = $1::UUID
ORDER BY rank;

-- name: UpsertDemoAccount :one
INSERT INTO demo_accounts (id, user_id, identifiers, email, phone, status, payload)
VALUES (
    sqlc.arg(id)::TEXT,
    sqlc.arg(user_id)::TEXT,
    sqlc.arg(identifiers)::TEXT[],
    sqlc.arg(email)::TEXT,
    sqlc.arg(phone)::TEXT,
    sqlc.arg(status)::VARCHAR(32),
    sqlc.arg(payload)::JSONB
)
ON CONFLICT (id) DO UPDATE
SET user_id = EXCLUDED.user_id,
    identifiers = EXCLUDED.identifiers,
    email = EXCLUDED.email,
    phone = EXCLUDED.phone,
    status = EXCLUDED.status,
    payload = EXCLUDED.payload,
    updated_at = now()
RETURNING *;

-- name: FindDemoAccountByIdentifier :one
SELECT * FROM demo_accounts
WHERE sqlc.arg(identifier)::TEXT = user_id
   OR sqlc.arg(identifier)::TEXT = ANY(identifiers)
LIMIT 1;

-- name: UpsertDemoBooking :one
INSERT INTO demo_bookings (
    id,
    booking_number,
    identifiers,
    service,
    master,
    booking_date,
    booking_time,
    status,
    payload
)
VALUES (
    sqlc.arg(id)::TEXT,
    sqlc.arg(booking_number)::TEXT,
    sqlc.arg(identifiers)::TEXT[],
    sqlc.arg(service)::TEXT,
    sqlc.arg(master)::TEXT,
    sqlc.narg(booking_date)::DATE,
    sqlc.narg(booking_time)::TIME,
    sqlc.arg(status)::VARCHAR(32),
    sqlc.arg(payload)::JSONB
)
ON CONFLICT (id) DO UPDATE
SET booking_number = EXCLUDED.booking_number,
    identifiers = EXCLUDED.identifiers,
    service = EXCLUDED.service,
    master = EXCLUDED.master,
    booking_date = EXCLUDED.booking_date,
    booking_time = EXCLUDED.booking_time,
    status = EXCLUDED.status,
    payload = EXCLUDED.payload,
    updated_at = now()
RETURNING *;

-- name: FindDemoBookingByIdentifier :one
SELECT * FROM demo_bookings
WHERE sqlc.arg(identifier)::TEXT = booking_number
   OR sqlc.arg(identifier)::TEXT = ANY(identifiers)
LIMIT 1;

-- name: UpsertDemoWorkspaceBooking :one
INSERT INTO demo_workspace_bookings (
    id,
    booking_number,
    identifiers,
    workspace_type,
    booking_date,
    booking_time,
    duration_hours,
    status,
    payload
)
VALUES (
    sqlc.arg(id)::TEXT,
    sqlc.arg(booking_number)::TEXT,
    sqlc.arg(identifiers)::TEXT[],
    sqlc.arg(workspace_type)::TEXT,
    sqlc.narg(booking_date)::DATE,
    sqlc.narg(booking_time)::TIME,
    sqlc.narg(duration_hours)::INT,
    sqlc.arg(status)::VARCHAR(32),
    sqlc.arg(payload)::JSONB
)
ON CONFLICT (id) DO UPDATE
SET booking_number = EXCLUDED.booking_number,
    identifiers = EXCLUDED.identifiers,
    workspace_type = EXCLUDED.workspace_type,
    booking_date = EXCLUDED.booking_date,
    booking_time = EXCLUDED.booking_time,
    duration_hours = EXCLUDED.duration_hours,
    status = EXCLUDED.status,
    payload = EXCLUDED.payload,
    updated_at = now()
RETURNING *;

-- name: FindDemoWorkspaceBookingByIdentifier :one
SELECT * FROM demo_workspace_bookings
WHERE sqlc.arg(identifier)::TEXT = booking_number
   OR sqlc.arg(identifier)::TEXT = ANY(identifiers)
LIMIT 1;

-- name: UpsertDemoPayment :one
INSERT INTO demo_payments (
    id,
    payment_id,
    identifiers,
    amount_cents,
    paid_at,
    status,
    purpose,
    payload
)
VALUES (
    sqlc.arg(id)::TEXT,
    sqlc.arg(payment_id)::TEXT,
    sqlc.arg(identifiers)::TEXT[],
    sqlc.arg(amount_cents)::INT,
    sqlc.narg(paid_at)::TIMESTAMPTZ,
    sqlc.arg(status)::VARCHAR(32),
    sqlc.arg(purpose)::TEXT,
    sqlc.arg(payload)::JSONB
)
ON CONFLICT (id) DO UPDATE
SET payment_id = EXCLUDED.payment_id,
    identifiers = EXCLUDED.identifiers,
    amount_cents = EXCLUDED.amount_cents,
    paid_at = EXCLUDED.paid_at,
    status = EXCLUDED.status,
    purpose = EXCLUDED.purpose,
    payload = EXCLUDED.payload,
    updated_at = now()
RETURNING *;

-- name: FindDemoPaymentByIdentifier :one
SELECT * FROM demo_payments
WHERE sqlc.arg(identifier)::TEXT = payment_id
   OR sqlc.arg(identifier)::TEXT = ANY(identifiers)
LIMIT 1;
