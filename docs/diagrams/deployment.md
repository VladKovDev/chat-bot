# Deployment Diagram

```mermaid
flowchart TB
  subgraph "docker compose"
    Postgres[(postgres: pgvector/pgvector:pg16)]
    Migrate[decision-migrate]
    NLP[nlp-service]
    Decision[decision-engine]
    Website[website]
    Mock[mock-external-services]
  end

  HostBrowser[Developer browser]
  Seeds[./seeds read-only mount]
  Contracts[decision-engine/contracts read-only mount]

  HostBrowser -->|http://localhost:8081| Website
  HostBrowser -->|optional http://localhost:8080/api/v1/ready| Decision
  HostBrowser -->|optional http://localhost:8082/ready| NLP
  HostBrowser -->|optional http://localhost:8090/health| Mock
  Migrate -->|goose migrations| Postgres
  Decision -->|SQL| Postgres
  Decision -->|HTTP nlp-service:8080| NLP
  Decision -->|HTTP mock-external-services:8090| Mock
  Decision -->|seed import| Seeds
  Mock -->|fixture-backed responses| Seeds
  Mock -->|contract reference| Contracts
  Website -->|HTTP decision-engine:8080| Decision
```

Default published ports are 5442 for Postgres, 8080 for decision-engine, 8081
for website, 8082 for NLP, and 8090 for mock external services. The website
container is the browser entry point; direct service ports are used by health
checks, E2E tests, and manual diagnostics.
