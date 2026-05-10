# Architecture

## Current Direction

The repository is being converged onto one greenfield runtime. The decision-engine schema and session lifecycle are defined for a clean database bootstrap first, not for compatibility with earlier DB layouts or earlier runtime assumptions.

## Session Model

- Target sessions are keyed by explicit identity: `channel + external_user_id` or `channel + client_id`.
- The target `sessions` table does not persist legacy `chat_id` or session summaries.
- `mode`, `active_topic`, `last_intent`, `fallback_count`, `operator_status`, and `metadata` are part of the base schema, not an upgrade overlay.
- Transition logs include `event` and `reason` in the base schema.

## Bootstrap Rules

- Empty PostgreSQL bootstrapping is the normal path.
- The first migration creates `users`, `sessions`, `messages`, `transitions_log`, and `actions_log` in their target shape.
- Session creation ensures the backing `users` row exists before inserting the session row, so a clean database can start serving traffic without seed/backfill steps.

## Legacy Boundary

- Old `chat_id=1` production assumptions are intentionally removed from the target model.
- `chat_id` is only accepted on the `dev-cli` request path, where it is translated into a stable external identity (`chat:<id>`).
- Old DB contents are not a supported migration target for this reset step.
