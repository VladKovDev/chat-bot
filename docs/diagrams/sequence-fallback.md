# Fallback Sequence

```mermaid
sequenceDiagram
  participant U as User browser
  participant W as Website adapter
  participant D as Decision engine
  participant N as NLP service
  participant P as PostgreSQL + pgvector

  U->>W: message.user with unknown or low-confidence text
  W->>D: POST /api/v1/messages
  D->>P: persist user message and load recent history
  D->>N: POST /api/v1/embed
  alt NLP is available but confidence is below threshold
    D->>P: log low_confidence decision and fallback candidates
    D->>P: increment session/session_context fallback_count
    D->>P: persist clarification bot message
    D-->>W: MessageResponse with clarification text and quick replies
    W-->>U: message.bot
  else NLP is unavailable
    D->>P: log fallback candidate evidence embedding_unavailable
    D->>P: persist controlled fallback bot message under latency budget
    D-->>W: MessageResponse with fallback text
    W-->>U: message.bot
  end
  opt repeated fallback crosses operator policy
    D->>P: create operator_queue(status=waiting, reason=low_confidence_repeated)
    D->>P: set session mode=waiting_operator/operator_status=waiting
    D-->>W: MessageResponse with handoff.status=waiting
    W-->>U: handoff.queued
  end
```

Fallback remains controlled and persisted. It does not call an LLM runtime and
does not silently drop the user message; E2E coverage asserts fallback evidence
and the under-three-second degraded NLP path.
