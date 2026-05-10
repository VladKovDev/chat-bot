# Component Diagram

```mermaid
flowchart LR
  Chat[Web chat]
  OperatorUI[Operator UI]
  Website[Website transport adapter]
  Decision[Decision engine]
  NLP[NLP service]
  DB[(PostgreSQL + pgvector)]
  Mock[Mock external services]
  Seeds[(Seed fixtures)]

  Chat <-->|WebSocket /ws| Website
  OperatorUI -->|HTTP /api/operator/*| Website
  Website -->|HTTP /api/v1/sessions<br/>/api/v1/messages<br/>/api/v1/operator/*| Decision
  Decision -->|HTTP /ready, /embed| NLP
  Decision -->|sessions, messages, decisions,<br/>actions, operators, semantic catalog| DB
  Decision -->|read-only provider lookups| Mock
  Mock -->|read-only fixture data| Seeds
  Decision -->|seed import on startup| Seeds
```

The decision engine owns session restoration, message persistence, decisions,
action execution, operator handoff state, readiness, and the HTTP v1 contract.
The website adapter owns the browser WebSocket contract and proxies operator UI
HTTP calls. The NLP service supplies 384-dimensional embeddings for semantic
intent and knowledge lookup. Mock external services expose read-only provider
contracts for booking, workspace booking, payment, user account, and pricing
scenarios; the demo never mutates those external fixtures.
