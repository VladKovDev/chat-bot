# Business Lookup Sequence

```mermaid
sequenceDiagram
  participant U as User browser
  participant W as Website adapter
  participant D as Decision engine
  participant N as NLP service
  participant P as PostgreSQL + pgvector
  participant M as Mock external services

  U->>W: message.user with booking/payment/workspace/account identifier
  W->>D: POST /api/v1/messages {type:"user_message", session_id, event_id}
  D->>P: load session and persist user message
  D->>N: POST /api/v1/embed
  D->>P: match intent_examples / knowledge_chunks
  D->>M: GET provider endpoint from mock-external-providers-v1
  M-->>D: found, not_found, invalid, or unavailable result
  D->>P: log decision_logs and decision_candidates
  D->>P: log actions_log with provider, status, duration_ms, redacted payload
  alt provider result is found
    D->>P: persist bot message rendered from provider values
    D-->>W: MessageResponse with response_key *_found and no handoff
    W-->>U: message.bot with fixture-backed details
  else provider result is not_found or invalid
    D->>P: persist controlled bot message and retry/operator quick replies
    D-->>W: MessageResponse with no internal provider details
    W-->>U: message.bot
  else provider result is unavailable
    D->>P: queue operator handoff with reason=business_error
    D->>P: persist controlled fallback bot message
    D-->>W: MessageResponse with handoff.status=waiting
    W-->>U: message.bot then handoff.queued
  end
```

The lookup action set is fixture-backed and read-only: `find_booking`,
`find_workspace_booking`, `find_payment`, and `find_user_account`. Provider
evidence is stored in `actions_log`; user-facing responses use safe text and do
not expose SQL, panic, or upstream internals.
