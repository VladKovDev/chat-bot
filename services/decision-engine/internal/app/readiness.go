package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	infranlp "github.com/VladKovDev/chat-bot/internal/infrastructure/nlp"
	"github.com/VladKovDev/chat-bot/internal/transport/http/handler"
	"github.com/jackc/pgx/v5"
)

const expectedMigrationVersion int64 = 20260510210000

type readinessDB interface {
	Ping(context.Context) error
	QueryRow(context.Context, string, ...any) pgx.Row
}

func NewReadinessProvider(db readinessDB, nlpConfig infranlp.EmbedderConfig) handler.ReadinessProvider {
	client := &http.Client{Timeout: nlpConfig.Timeout}
	if client.Timeout <= 0 {
		client.Timeout = 5 * time.Second
	}
	nlpBaseURL := strings.TrimRight(strings.TrimSpace(nlpConfig.BaseURL), "/")

	return func(ctx context.Context) handler.ReadyResponse {
		checks := map[string]handler.ReadinessItem{
			"database":        checkDatabase(ctx, db),
			"migrations":      checkMigrations(ctx, db),
			"nlp":             checkNLP(ctx, client, nlpBaseURL, nlpConfig.ExpectedDimension),
			"operator_tables": checkOperatorTables(ctx, db),
			"pgvector":        checkPgvector(ctx, db),
			"seed_data":       checkSeedData(ctx, db),
		}

		ready := true
		for _, check := range checks {
			if !check.Ready {
				ready = false
				break
			}
		}

		return handler.ReadyResponse{
			Ready:     ready,
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Checks:    checks,
		}
	}
}

func checkDatabase(ctx context.Context, db readinessDB) handler.ReadinessItem {
	if db == nil {
		return notReady("database pool is not initialized")
	}
	if err := db.Ping(ctx); err != nil {
		return notReady(fmt.Sprintf("database ping failed: %v", err))
	}
	return ready("database ping ok")
}

func checkMigrations(ctx context.Context, db readinessDB) handler.ReadinessItem {
	if db == nil {
		return notReady("database pool is not initialized")
	}

	var version int64
	err := db.QueryRow(ctx, `
SELECT version_id
FROM goose_db_version
WHERE is_applied = true
ORDER BY version_id DESC
LIMIT 1`).Scan(&version)
	if err != nil {
		return notReady(fmt.Sprintf("migration version unavailable: %v", err))
	}
	if version < expectedMigrationVersion {
		return notReady(fmt.Sprintf("migration version %d is below expected %d", version, expectedMigrationVersion))
	}
	return ready(fmt.Sprintf("migration version %d applied", version))
}

func checkNLP(ctx context.Context, client *http.Client, baseURL string, expectedDimension int) handler.ReadinessItem {
	if baseURL == "" {
		return notReady("nlp base_url is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/ready", nil)
	if err != nil {
		return notReady(fmt.Sprintf("nlp readiness request failed: %v", err))
	}

	resp, err := client.Do(req)
	if err != nil {
		return notReady(fmt.Sprintf("nlp readiness request failed: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return notReady(fmt.Sprintf("nlp readiness body failed: %v", err))
	}
	if resp.StatusCode != http.StatusOK {
		return notReady(fmt.Sprintf("nlp readiness status %d: %s", resp.StatusCode, strings.TrimSpace(string(body))))
	}

	var payload struct {
		Status    string `json:"status"`
		Model     string `json:"model"`
		Dimension int    `json:"dimension"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return notReady(fmt.Sprintf("nlp readiness payload invalid: %v", err))
	}
	if payload.Status != "ready" {
		return notReady(fmt.Sprintf("nlp status is %q", payload.Status))
	}
	if expectedDimension > 0 && payload.Dimension != expectedDimension {
		return notReady(fmt.Sprintf("nlp dimension %d is not expected %d", payload.Dimension, expectedDimension))
	}

	return ready(fmt.Sprintf("nlp ready: %s dimension=%d", payload.Model, payload.Dimension))
}

func checkPgvector(ctx context.Context, db readinessDB) handler.ReadinessItem {
	if db == nil {
		return notReady("database pool is not initialized")
	}

	var extVersion string
	err := db.QueryRow(ctx, `
SELECT extversion
FROM pg_extension
WHERE extname = 'vector'`).Scan(&extVersion)
	if err != nil {
		return notReady(fmt.Sprintf("pgvector extension unavailable: %v", err))
	}
	return ready(fmt.Sprintf("pgvector extension installed: %s", extVersion))
}

func checkOperatorTables(ctx context.Context, db readinessDB) handler.ReadinessItem {
	if db == nil {
		return notReady("database pool is not initialized")
	}

	var operatorsTable, queueTable, assignmentsTable, eventsTable bool
	err := db.QueryRow(ctx, `
SELECT
  to_regclass('public.operators') IS NOT NULL,
  to_regclass('public.operator_queue') IS NOT NULL,
  to_regclass('public.operator_assignments') IS NOT NULL,
  to_regclass('public.operator_events') IS NOT NULL`).Scan(
		&operatorsTable,
		&queueTable,
		&assignmentsTable,
		&eventsTable,
	)
	if err != nil {
		return notReady(fmt.Sprintf("operator table check unavailable: %v", err))
	}

	missing := make([]string, 0, 4)
	if !operatorsTable {
		missing = append(missing, "operators")
	}
	if !queueTable {
		missing = append(missing, "operator_queue")
	}
	if !assignmentsTable {
		missing = append(missing, "operator_assignments")
	}
	if !eventsTable {
		missing = append(missing, "operator_events")
	}
	if len(missing) > 0 {
		return notReady("missing operator tables: " + strings.Join(missing, ", "))
	}

	var operators int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM operators`).Scan(&operators); err != nil {
		return notReady(fmt.Sprintf("operator seed check unavailable: %v", err))
	}
	if operators == 0 {
		return notReady("missing operator seed data: operators")
	}

	return ready(fmt.Sprintf("operator tables ready: operators=%d", operators))
}

func checkSeedData(ctx context.Context, db readinessDB) handler.ReadinessItem {
	if db == nil {
		return notReady("database pool is not initialized")
	}

	var intents, examples, articles, chunks, accounts, bookings, workspaceBookings, payments int
	err := db.QueryRow(ctx, `
SELECT
  (SELECT COUNT(*) FROM intents WHERE active = true),
  (SELECT COUNT(*) FROM intent_examples WHERE active = true),
  (SELECT COUNT(*) FROM knowledge_articles),
  (SELECT COUNT(*) FROM knowledge_chunks),
  (SELECT COUNT(*) FROM demo_accounts),
  (SELECT COUNT(*) FROM demo_bookings),
  (SELECT COUNT(*) FROM demo_workspace_bookings),
  (SELECT COUNT(*) FROM demo_payments)`).Scan(
		&intents,
		&examples,
		&articles,
		&chunks,
		&accounts,
		&bookings,
		&workspaceBookings,
		&payments,
	)
	if err != nil {
		return notReady(fmt.Sprintf("seed data unavailable: %v", err))
	}

	missing := make([]string, 0, 8)
	if intents == 0 {
		missing = append(missing, "intent catalog")
	}
	if examples == 0 {
		missing = append(missing, "intent examples")
	}
	if articles == 0 {
		missing = append(missing, "knowledge articles")
	}
	if chunks == 0 {
		missing = append(missing, "knowledge chunks")
	}
	if accounts == 0 {
		missing = append(missing, "demo accounts")
	}
	if bookings == 0 {
		missing = append(missing, "demo bookings")
	}
	if workspaceBookings == 0 {
		missing = append(missing, "demo workspace bookings")
	}
	if payments == 0 {
		missing = append(missing, "demo payments")
	}
	if len(missing) > 0 {
		return notReady("missing seed data: " + strings.Join(missing, ", "))
	}

	return ready(fmt.Sprintf(
		"seed data ready: intents=%d examples=%d knowledge_articles=%d knowledge_chunks=%d demo_accounts=%d demo_bookings=%d demo_workspace_bookings=%d demo_payments=%d",
		intents,
		examples,
		articles,
		chunks,
		accounts,
		bookings,
		workspaceBookings,
		payments,
	))
}

func ready(message string) handler.ReadinessItem {
	return handler.ReadinessItem{Ready: true, Message: message}
}

func notReady(message string) handler.ReadinessItem {
	return handler.ReadinessItem{Ready: false, Message: message}
}
