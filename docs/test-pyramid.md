# Test Pyramid And Core Gate

`chat-bot-523.21` establishes a deterministic local core gate for the BRD implementation path. It is a smoke/core gate, not the full E2E matrix.

## Core Gate

Run from the repository root:

```bash
make check-core
```

Equivalent direct command:

```bash
./scripts/check-core.sh
```

The gate is non-interactive and does not start long-lived services. It runs:

1. `git diff --check`
2. JSON/YAML parse validation for repository config, contracts, and seeds
3. static legacy-runtime grep over active runtime paths to reject `/llm/decide`, `llmClient.Decide`, `Ollama`, and `GigaChat`
4. `cd services/decision-engine && go test -count=1 ./...`
5. `cd services/nlp-service && uv run --extra test python -m pytest`
6. `cd services/transport-adapters/website/backend && go test -count=1 ./...`
7. `node --test services/transport-adapters/website/frontend/assets/js/*.test.mjs`

## Layers

| Layer | Current coverage | Core gate |
| --- | --- | --- |
| Unit | decision policy, response selection, renderer, domain session/intent rules, provider/action behavior, frontend render helpers | included |
| Contract | decision-engine HTTP v1 document checks, WebSocket event contract checks, NLP `/health`/`/ready`/`/preprocess`/`/embed`, mock external provider contract | included |
| Integration | `/api/v1/messages` handler with deterministic fake semantic matcher, website backend HTTP/WS adapters against `httptest`, worker persistence orchestration with fake transaction boundary | included |
| Migration/bootstrap | static migration and sqlc query coverage for pgcrypto/vector, BRD catalog, DB bootstrap tables, operator queue, indexes, FK, session versioning | included as static tests |
| Security regressions | public error masking, WebSocket Origin allowlist, DOM text rendering/no `innerHTML`, legacy LLM runtime absence | included |
| Full E2E | E2E-001 through E2E-038 over DB/NLP/decision/website/operator/mock providers | excluded from `check-core`; run with `make e2e-smoke` or `make e2e-full` |

## BRD Surface Map

| Surface | Current proof |
| --- | --- |
| Semantic matcher | `services/decision-engine/internal/app/decision/semantic_matcher_test.go`, `services/decision-engine/internal/infrastructure/repository/postgres/semantic_catalog_integration_test.go`, and E2E semantic flows cover exact-command fallback, pgvector search, thresholds, ambiguity, and candidate persistence. |
| Preprocessing/NLP | `services/nlp-service/tests/test_contract.py` covers Russian preprocessing, lemmas, deterministic embeddings, batch embeddings, unavailable model behavior, and request bounds. |
| Renderer | `services/decision-engine/internal/app/presenter/render_test.go`, frontend `chat.test.mjs`, and `operator.test.mjs`. |
| Decision policy | `services/decision-engine/internal/app/decision/service_test.go`, `processor/response_selector_test.go`, and handler message tests. |
| Providers | `services/decision-engine/internal/app/provider/mock_services_test.go`, action tests, seed validation, and `contracts/mock-external-providers-v1.json`. |
| HTTP contracts | `services/decision-engine/contracts/http-v1.json` checked by handler tests. |
| WS/operator contracts | `services/transport-adapters/website/contracts/websocket.json`, backend websocket tests, operator UI tests, and operator service tests. |
| DB bootstrap/migrations | `services/decision-engine/internal/infrastructure/repository/postgres/*_schema_test.go`. |
| Transactional persistence | `services/decision-engine/internal/app/worker/message_worker_test.go` covers commit and rollback behavior for messages, decisions, actions, transitions, and session context. |

## Optional Live DB Gate

Live pgvector migration remains outside `make check-core` because it requires Docker/PostgreSQL and a disposable database. Use it as the DB smoke gate when the environment is available:

```bash
cd services/decision-engine
DATABASE_URL='postgres://postgres:postgres@localhost:5442/chat-bot?sslmode=disable' make migrate-up
```

For release completion, run the full E2E matrix separately from this smoke/core gate.

## Compose And E2E Gates

Root compose smoke:

```bash
scripts/smoke-compose.sh
```

This starts the root `docker-compose.yml`, waits for `GET /api/v1/ready`, prints
the readiness payload on success, and removes containers/volumes unless
`KEEP_COMPOSE=1` is set.

Playwright smoke E2E:

```bash
make e2e-smoke
npm run test:e2e:smoke
```

Full E2E:

```bash
make e2e-full
npm run test:e2e:full
```

The E2E matrix is implemented in `tests/e2e/full-matrix.spec.ts` and contains
E2E-001 through E2E-038. Reports and failure artifacts are written to
`tests/e2e/test-results/e2e-html/`, `tests/e2e/test-results/e2e-results.json`,
and `tests/e2e/test-results/e2e-artifacts/`.
