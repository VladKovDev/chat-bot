# Architecture

## Current Direction

The repository is being converged onto one greenfield runtime. The decision-engine schema and session lifecycle are defined for a clean database bootstrap first, not for compatibility with earlier DB layouts or earlier runtime assumptions.

## Runtime Overview

The root demo runtime is:

1. website chat/operator UI
2. decision-engine HTTP API and worker pipeline
3. Python NLP service for deterministic demo embeddings
4. PostgreSQL with pgvector for sessions, messages, decisions, operators, demo provider data, intent examples, and knowledge chunks
5. fixture-backed mock external providers for booking, workspace, payment, account, and pricing lookups

The runtime is documented by the root [README](../README.md), diagrams under [docs/diagrams](diagrams), and versioned contracts:

- `services/decision-engine/contracts/http-v1.json`
- `services/transport-adapters/website/contracts/websocket.json`
- `services/decision-engine/contracts/mock-external-providers-v1.json`

The demo uses deterministic fake hash embeddings in the NLP service. That keeps local and CI tests reproducible; it is not a production model integration.

## Session Model

- Target sessions are keyed by explicit identity: `channel + external_user_id` or `channel + client_id`.
- The target `sessions` table does not persist legacy `chat_id` or session summaries.
- `mode`, `active_topic`, `last_intent`, `fallback_count`, `operator_status`, and `metadata` are part of the base schema, not an upgrade overlay.
- Transition logs include `event` and `reason` in the base schema.

## Bootstrap Rules

- Empty PostgreSQL bootstrapping is the normal path.
- The first migration creates `users`, `sessions`, `messages`, `transitions_log`, and `actions_log` in their target shape.
- Session creation ensures the backing `users` row exists before inserting the session row, so a clean database can start serving traffic without seed/backfill steps.
- The compose startup path runs Goose migrations, starts NLP, then lets decision-engine seed demo provider data plus semantic intent/knowledge data from `seeds/`.
- `GET /api/v1/ready` is the operational gate. It checks DB, latest migration, pgvector, NLP readiness/dimension, operator tables and operator seed data, and semantic/demo seed data.

## Legacy Boundary

- Old `chat_id=1` production assumptions are intentionally removed from the target model.
- The target HTTP contract rejects legacy `chat_id`; clients create or resume sessions through `/api/v1/sessions` with `channel + client_id` or `channel + external_user_id`.
- Old DB contents are not a supported migration target for this reset step.
- The demo does not use real payment, CRM, booking, account, or pricing systems. Those boundaries are read-only mock providers with documented contracts and fixture data.
