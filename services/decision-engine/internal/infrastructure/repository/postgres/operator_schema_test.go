package postgres

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOperatorQueueMigrationDocumentsLifecycleConstraints(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "..", "..", "migrations", "20260510193000_operator_queue.sql")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read operator queue migration: %v", err)
	}
	sql := string(data)

	required := []string{
		"CREATE TABLE IF NOT EXISTS operator_queue",
		"CHECK (status IN ('waiting', 'accepted', 'closed'))",
		"CHECK (reason IN ('manual_request', 'low_confidence_repeated', 'complaint', 'business_error'))",
		"CREATE TABLE IF NOT EXISTS operator_assignments",
		"CREATE TABLE IF NOT EXISTS operator_events",
		"CREATE INDEX idx_operator_queue_status_created_at",
		"CREATE INDEX idx_operator_queue_assigned_operator_status",
		"CREATE UNIQUE INDEX idx_operator_queue_open_session",
	}
	for _, fragment := range required {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
}
