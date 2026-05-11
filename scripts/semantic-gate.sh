#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DECISION_DIR="$ROOT_DIR/services/decision-engine"
DECISION_SEMANTIC_TESTS='Test(Semantic|ActualIntentCatalog|CatalogMatcher|DecisionServiceAppliesSemanticThresholdAndAmbiguityPolicy)'
POSTGRES_SEMANTIC_TESTS='Test(BRDPgvectorCatalogMigrationDocumentsTargetSchema|SemanticCatalogRepository(SearchKnowledgeChunksRejectsDimensionMismatch|RejectsEmbeddingDimensionMismatch))'

run() {
  printf '\n==> %s\n' "$*"
  "$@"
}

cd "$DECISION_DIR"

run go test -count=1 ./internal/app/decision -run "$DECISION_SEMANTIC_TESTS"
run go test -count=1 ./internal/infrastructure/repository/postgres -run "$POSTGRES_SEMANTIC_TESTS"
run go run ./cmd/semantic-gate "$@"
