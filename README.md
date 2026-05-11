# Chat Bot

Greenfield BRD demo for a beauty-coworking support bot. The current runtime is
the demo path, not the old LLM-first path: website chat and operator UI talk to
the Go decision engine, the decision engine uses PostgreSQL + pgvector and the
Python NLP embedding service, and business lookups use deterministic mock
external providers.

## What You Can Run

From a fresh clone with Docker, Go, Node/npm, Python `uv`, Ruby, and Make
available:

```bash
cp .env.example .env
docker compose up --build
```

The compose stack starts:

| Service | Default URL | Purpose |
| --- | --- | --- |
| `postgres` | `localhost:5442` | PostgreSQL 16 with pgvector |
| `decision-migrate` | internal one-shot | Goose migrations for an empty DB volume |
| `nlp-service` | `http://localhost:8082` | deterministic fake hash embeddings |
| `decision-engine` | `http://localhost:8080` | HTTP API, readiness, decision flow, operator APIs |
| `website` | `http://localhost:8081` | demo web chat and operator UI |
| `mock-external-services` | `http://localhost:8090` | fixture-backed provider endpoints |

The demo readiness gate is:

```bash
curl -fsS http://localhost:8080/api/v1/ready
```

`/api/v1/ready` returns 503 until these checks pass: DB ping, latest Goose
migration, pgvector extension, NLP readiness and dimension, operator tables and
operator seed data, and semantic/demo seed data. It returns 200 only after the
empty-volume bootstrap has completed.

For a one-command compose smoke with automatic cleanup:

```bash
scripts/smoke-compose.sh
```

Set `KEEP_COMPOSE=1 scripts/smoke-compose.sh` when you want to keep the stack
for manual inspection after the smoke.

## Seed Data

The decision engine loads demo data from `seeds/` on startup:

- `intents.json` for intent definitions, examples, response keys, and quick replies
- `knowledge-base.json` for FAQ/pricing/rules knowledge chunks
- `demo-bookings.json`, `demo-workspace-bookings.json`, `demo-payments.json`,
  and `demo-users.json` for read-only lookup fixtures
- `demo-operators.json` for the operator queue demo
- `mock-external-services.json` for provider error/success fixture behavior

In compose, `./seeds` is mounted read-only at `/app/seeds`. Startup seeds the
demo provider tables, then embeds intent examples and knowledge chunks through
the NLP service into pgvector-backed catalog tables.

## Quality Gates

Run the focused semantic gate from the repository root when you need quick
semantic matcher/catalog/corpus proof without Docker or the full runtime:

```bash
make semantic-gate
```

Equivalent direct command:

```bash
./scripts/semantic-gate.sh
```

Run the core deterministic gate from the repository root:

```bash
make check-core
```

Equivalent direct command:

```bash
./scripts/check-core.sh
```

The current verification surfaces are:

| Command | Proof mode | What it proves |
| --- | --- | --- |
| `make semantic-gate` / `./scripts/semantic-gate.sh` | deterministic semantic gate; no Docker/services | runs focused decision-engine semantic tests for matcher behavior, catalog matcher heuristics, semantic threshold policy, and static semantic catalog dimension/schema guards, then prints `go run ./cmd/semantic-gate` corpus metrics against `seeds/intents.json` and `internal/app/decision/testdata/semantic_gold_corpus.json`. |
| `make check-core` / `./scripts/check-core.sh` | broader deterministic core gate; no Docker/services | runs `git diff --check`, JSON/YAML parse validation, legacy-runtime grep, and the Go/Python/Node unit+contract+integration suites. |
| `scripts/smoke-compose.sh` | live local compose smoke | boots the root compose stack, waits for `GET /api/v1/ready`, and proves bootstrap/readiness for local Postgres, decision-engine, website, NLP, and mock external services. |
| `make e2e-smoke` / `npm run test:e2e:smoke` | live local Playwright smoke | runs the `@smoke` subset of the matrix against an ephemeral compose stack. |
| `make e2e-full` / `npm run test:e2e:full` | live local Playwright full matrix | runs the full Playwright matrix against an ephemeral compose stack. |

Evidence boundaries matter here:

- `semantic-gate` is the standalone quick semantic proof. It is deterministic and non-interactive, but it does not boot Postgres, NLP, website, or browser surfaces.
- `check-core` is mock-only/static proof. It does not boot services or Docker.
- `scripts/smoke-compose.sh`, `make e2e-smoke`, and `make e2e-full` are live-runtime proof for the local stack because they boot containers and hit real local HTTP/UI surfaces.
- None of the commands above are real-Qwen or live third-party proof. The default runtime still uses deterministic fake hash embeddings and fixture-backed mock external providers.

Smoke E2E:

```bash
make e2e-smoke
npm run test:e2e:smoke
```

Full E2E matrix:

```bash
make e2e-full
npm run test:e2e:full
```

The E2E suite lives under `tests/e2e`. The current matrix spans E2E-001 through
E2E-039 in `tests/e2e/full-matrix.spec.ts`, which expands to 44 Playwright
cases because some IDs have suffixed variants such as `E2E-001b`,
`E2E-001c`, `E2E-006b`, `E2E-009b`, and `E2E-009c`. The smoke command selects
the 13 cases tagged `@smoke`. By default the suite uses its own compose project
and ports:

| Env | Default |
| --- | --- |
| `E2E_COMPOSE_PROJECT` | `chat-bot-e2e` |
| `E2E_POSTGRES_PORT` | `15442` |
| `E2E_DECISION_PORT` | `18080` |
| `E2E_WEBSITE_PORT` | `18081` |
| `E2E_NLP_PORT` | `18082` |
| `E2E_MOCK_EXTERNAL_PORT` | `18090` |

Use `E2E_SKIP_COMPOSE=1` to point the tests at an already running stack, and
`E2E_KEEP_COMPOSE=1` to keep containers after a run. Test reports and failure
artifacts are written under:

- `tests/e2e/test-results/e2e-html/`
- `tests/e2e/test-results/e2e-results.json`
- `tests/e2e/test-results/e2e-artifacts/`

## Architecture And Contracts

Start with:

- [docs/architecture.md](docs/architecture.md) for the runtime reset and session/bootstrap rules
- [docs/test-pyramid.md](docs/test-pyramid.md) for the core gate and E2E split
- [docs/development-plan-from-brd.md](docs/development-plan-from-brd.md) for the BRD source map

Diagrams:

- [Component diagram](docs/diagrams/component.md)
- [ERD](docs/diagrams/erd.md)
- [Deployment diagram](docs/diagrams/deployment.md)
- [User message sequence](docs/diagrams/sequence-user-message.md)
- [Operator handoff sequence](docs/diagrams/sequence-operator-handoff.md)

Versioned contracts:

- `services/decision-engine/contracts/http-v1.json` documents decision-engine
  HTTP routes, public errors, sessions, messages, readiness, and operator APIs.
- `services/transport-adapters/website/contracts/websocket.json` documents
  website client/server WebSocket events.
- `services/decision-engine/contracts/mock-external-providers-v1.json`
  documents fixture-backed booking, workspace, payment, account, and pricing
  provider shapes.

The public demo API surface is rooted at `/api/v1`: `/health`, `/ready`,
`/sessions`, `/messages`, `/sessions/{session_id}/messages`,
`/domain/schema`, and operator queue/session routes.

## Demo Limits

This is a deterministic demo system. It is intentionally not wired to real
payment, CRM, booking, account, or pricing systems.

- External business providers are mocks backed by repository fixtures.
- Lookup data is read-only demo data; no real booking, payment, CRM, account, or
  workspace mutation is performed.
- The NLP service uses deterministic fake hash embeddings in demo/test mode, not
  a production embedding model.
- The runtime must not require `services/llm`, Ollama, or GigaChat.
- Provider failures are expected to produce safe public fallback/operator paths,
  not raw internal errors.

## Readiness Troubleshooting

If `docker compose up --build` starts but readiness stays 503, inspect the
failed check in the `/api/v1/ready` payload first:

```bash
curl -s http://localhost:8080/api/v1/ready | jq .
```

Useful follow-up commands:

```bash
docker compose ps
docker compose logs --no-color --tail=200 decision-migrate decision-engine nlp-service postgres
docker compose config
```

Common readiness failures:

| Check | Meaning | Usual fix |
| --- | --- | --- |
| `database` | decision engine cannot ping Postgres | check `postgres` health and `POSTGRES_*` env/ports |
| `migrations` | Goose version is missing or below `20260510210000` | recreate the demo volume or inspect `decision-migrate` logs |
| `pgvector` | `vector` extension is missing | use the root `pgvector/pgvector:pg16` compose service |
| `nlp` | NLP `/ready` failed or dimension is not 384 | inspect `nlp-service` logs and `NLP_EXPECTED_DIMENSION` |
| `operator_tables` | operator queue tables or operator seed data are missing | inspect migrations and `seeds/demo-operators.json` |
| `seed_data` | semantic/demo fixtures are missing | inspect `seeds/`, startup seed logs, and NLP embed availability |

For stale local state, reset only the compose demo project you are using:

```bash
docker compose down -v --remove-orphans
docker compose up --build
```

For E2E failures, open the HTML report under
`tests/e2e/test-results/e2e-html/` and inspect traces, screenshots, and videos
under `tests/e2e/test-results/e2e-artifacts/`.
