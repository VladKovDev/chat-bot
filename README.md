# Chat Bot

This repository is on the greenfield BRD implementation path.

## Greenfield Reset

- The source of truth is `docs/development-plan-from-brd.md`.
- Old database contents and old runtime compatibility are not supported.
- Fresh migrations define the target schema directly; there is no backfill path for legacy `chat_id=1` data.
- Production/runtime session identity is `channel + external_user_id` or `channel + client_id`.
- `chat_id` remains a `dev-cli` convenience only and is mapped to a stable session identity instead of being persisted in the target session model.

## Empty DB Bootstrap

1. Create an empty PostgreSQL database.
2. Run migrations from `services/decision-engine`:

```bash
make migrate-up
```

3. Generate SQL bindings after query changes:

```bash
sqlc generate
```

4. Build or test the service:

```bash
go build ./cmd/app
go test ./internal/domain/session ./internal/transport/http/handler
```

## Architecture

See [docs/architecture.md](docs/architecture.md) for the runtime reset decision and the current session/bootstrap rules.
