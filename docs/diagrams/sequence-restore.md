# Restore Sequence

```mermaid
sequenceDiagram
  participant U as User browser
  participant W as Website adapter
  participant D as Decision engine
  participant P as PostgreSQL

  U->>W: session.start {client_id}
  W->>D: POST /api/v1/sessions {channel:"website", client_id}
  D->>P: find active sessions row by channel/client_id
  alt active session exists
    P-->>D: existing session with active_topic, mode, operator_status, version
    D->>P: read persisted session_context
    D-->>W: StartSessionResponse {session_id, mode, active_topic, resumed:true}
    W-->>U: session.started with same session_id
  else no active session exists
    D->>P: create users row and sessions row
    D->>P: trigger creates session_context
    D-->>W: StartSessionResponse {session_id, mode:"standard", resumed:false}
    W-->>U: session.started with new session_id
  end
  U->>W: chat history/operator monitor resumes for session_id
  W->>D: GET /api/v1/sessions/{session_id}/messages
  D->>P: read persisted messages
  D-->>W: SessionMessagesResponse {items}
```

Restart recovery relies on Postgres state, not in-memory website state. The E2E
restore flow restarts `decision-engine` and `website`, starts a session with the
same `client_id`, and expects the same `session_id`, `resumed:true`, unchanged
message count, and preserved `active_topic`.
