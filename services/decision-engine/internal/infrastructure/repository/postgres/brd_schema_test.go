package postgres

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBRDPgvectorCatalogMigrationDocumentsTargetSchema(t *testing.T) {
	t.Parallel()

	sql := strings.Join([]string{
		readMigrationForTest(t, "20260510210000_brd_pgvector_catalog.sql"),
		readMigrationForTest(t, "20260511093000_semantic_catalog_dimension_guard.sql"),
		readMigrationForTest(t, "20260511120000_decision_candidate_source_alignment.sql"),
		readQueryForTest(t, "brd_catalog.sql"),
	}, "\n")

	required := []string{
		"CREATE EXTENSION IF NOT EXISTS vector",
		"ADD COLUMN IF NOT EXISTS idempotency_key TEXT",
		"ADD COLUMN IF NOT EXISTS detected_intent VARCHAR(80)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_session_idempotency_key",
		"CREATE TABLE IF NOT EXISTS session_context",
		"version = session_context.version + 1",
		"CREATE TABLE IF NOT EXISTS message_events",
		"UNIQUE (channel, external_event_id)",
		"CREATE TABLE IF NOT EXISTS intents",
		"CREATE TABLE IF NOT EXISTS intent_examples",
		"embedding vector(384) NOT NULL",
		"CREATE TABLE IF NOT EXISTS semantic_catalog_settings",
		"VALUES ('embedding_dimension', '384')",
		"CREATE INDEX IF NOT EXISTS idx_intent_examples_embedding_hnsw",
		"USING hnsw (embedding vector_cosine_ops)",
		"CREATE TABLE IF NOT EXISTS knowledge_articles",
		"CREATE TABLE IF NOT EXISTS knowledge_chunks",
		"CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_embedding_hnsw",
		"CREATE TABLE IF NOT EXISTS quick_replies",
		"CREATE TABLE IF NOT EXISTS decision_candidates",
		"'lexical_fuzzy'",
		"CREATE TABLE IF NOT EXISTS demo_accounts",
		"CREATE TABLE IF NOT EXISTS demo_bookings",
		"CREATE TABLE IF NOT EXISTS demo_workspace_bookings",
		"CREATE TABLE IF NOT EXISTS demo_payments",
		"ADD COLUMN IF NOT EXISTS message_id UUID REFERENCES messages(id) ON DELETE SET NULL",
		"ADD COLUMN IF NOT EXISTS provider VARCHAR(80) NOT NULL DEFAULT ''",
		"ADD COLUMN IF NOT EXISTS actor_type VARCHAR(16) NOT NULL DEFAULT 'system'",
	}
	for _, fragment := range required {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("BRD catalog migration missing %q", fragment)
		}
	}
}

func TestFreshSchemaMigrationSetCoversBRDTargetObjects(t *testing.T) {
	t.Parallel()

	sql := strings.Join([]string{
		readMigrationForTest(t, "20260216002310_init_tables.sql"),
		readMigrationForTest(t, "20260510170000_decision_logs.sql"),
		readMigrationForTest(t, "20260510193000_operator_queue.sql"),
		readMigrationForTest(t, "20260510210000_brd_pgvector_catalog.sql"),
		readMigrationForTest(t, "20260511093000_semantic_catalog_dimension_guard.sql"),
		readMigrationForTest(t, "20260511120000_decision_candidate_source_alignment.sql"),
		readQueryForTest(t, "sessions.sql"),
	}, "\n")

	required := []string{
		"CREATE EXTENSION IF NOT EXISTS pgcrypto",
		"CREATE EXTENSION IF NOT EXISTS vector",
		"CREATE TABLE IF NOT EXISTS users",
		"CREATE TABLE IF NOT EXISTS operators",
		"CREATE TABLE IF NOT EXISTS \"sessions\"",
		"CREATE TABLE IF NOT EXISTS session_context",
		"CREATE TABLE IF NOT EXISTS messages",
		"CREATE TABLE IF NOT EXISTS message_events",
		"CREATE TABLE IF NOT EXISTS intents",
		"CREATE TABLE IF NOT EXISTS intent_examples",
		"CREATE TABLE IF NOT EXISTS semantic_catalog_settings",
		"CREATE TABLE IF NOT EXISTS knowledge_articles",
		"CREATE TABLE IF NOT EXISTS knowledge_chunks",
		"CREATE TABLE IF NOT EXISTS quick_replies",
		"CREATE TABLE IF NOT EXISTS decision_logs",
		"CREATE TABLE IF NOT EXISTS decision_candidates",
		"CREATE TABLE IF NOT EXISTS demo_accounts",
		"CREATE TABLE IF NOT EXISTS demo_bookings",
		"CREATE TABLE IF NOT EXISTS demo_workspace_bookings",
		"CREATE TABLE IF NOT EXISTS demo_payments",
		"CREATE TABLE IF NOT EXISTS operator_queue",
		"CREATE TABLE IF NOT EXISTS operator_assignments",
		"CREATE TABLE IF NOT EXISTS operator_events",
		"CREATE TABLE IF NOT EXISTS actions_log",
		"CREATE TABLE IF NOT EXISTS transitions_log",
		"CREATE UNIQUE INDEX idx_session_active_external_user",
		"CREATE UNIQUE INDEX idx_session_active_client",
		"\"version\" = \"version\" + 1",
		"REFERENCES users(id)",
		"REFERENCES \"sessions\"(id)",
		"REFERENCES messages(id)",
		"REFERENCES intents(id)",
		"REFERENCES knowledge_articles(id)",
		"REFERENCES decision_logs(id)",
	}
	for _, fragment := range required {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("fresh schema migration set missing %q", fragment)
		}
	}
}

func readMigrationForTest(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "..", "..", "..", "migrations", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", name, err)
	}
	return string(data)
}

func readQueryForTest(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("queries", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read query %s: %v", name, err)
	}
	return string(data)
}
