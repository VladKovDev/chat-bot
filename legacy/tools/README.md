# Legacy tools

This directory contains historical project material that is intentionally
excluded from the BRD runtime.

- `llm-service/` preserves the old generative LLM demo service for reference.
  It must not be wired into Docker Compose, active handlers, or E2E flows.
- `decision-engine-legacy-rules/` preserves the old rule/transition classifier
  sources and JSON configs. The active runtime source of truth is the semantic
  catalog, deterministic decision service, and persisted session/operator model.
