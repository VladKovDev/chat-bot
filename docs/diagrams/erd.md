# ERD

```mermaid
erDiagram
  users ||--o{ sessions : owns
  sessions ||--|| session_context : mirrors
  sessions ||--o{ messages : contains
  sessions ||--o{ message_events : deduplicates
  messages ||--o{ message_events : may_reference
  sessions ||--o{ decision_logs : records
  messages ||--o{ decision_logs : decided_from
  decision_logs ||--o{ decision_candidates : ranks
  intents ||--o{ decision_candidates : may_reference
  sessions ||--o{ actions_log : records
  messages ||--o{ actions_log : may_reference
  sessions ||--o{ transitions_log : records
  messages ||--o{ transitions_log : may_reference
  sessions ||--o{ operator_queue : may_create
  users ||--o{ operator_queue : owns
  operator_queue ||--o{ operator_assignments : receives
  operators ||--o{ operator_assignments : accepts
  operator_queue ||--o{ operator_events : emits
  sessions ||--o{ operator_events : emits
  intents ||--o{ intent_examples : has
  intents ||--o{ quick_replies : offers
  knowledge_articles ||--o{ knowledge_chunks : has
  demo_accounts {
    text id PK
    text user_id UK
    text_array identifiers
    text email
    text phone
    varchar status
    jsonb payload
  }
  demo_bookings {
    text id PK
    text booking_number UK
    text_array identifiers
    text service
    text master
    date booking_date
    time booking_time
    varchar status
    jsonb payload
  }
  demo_workspace_bookings {
    text id PK
    text booking_number UK
    text_array identifiers
    text workspace_type
    date booking_date
    time booking_time
    int duration_hours
    varchar status
    jsonb payload
  }
  demo_payments {
    text id PK
    text payment_id UK
    text_array identifiers
    int amount_cents
    timestamptz paid_at
    varchar status
    text purpose
    jsonb payload
  }
```

The exact table definitions are in `services/decision-engine/migrations/`.
`intent_examples` and `knowledge_chunks` store `vector(384)` embeddings with
HNSW cosine indexes. `session_context` is kept in sync from `sessions` by a
trigger and is also updated by runtime context decisions. Demo provider tables
are independent fixture-backed lookup tables; the migrations do not define a
shared demo user table.
