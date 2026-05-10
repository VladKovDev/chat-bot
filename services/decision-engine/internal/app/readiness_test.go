package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	infranlp "github.com/VladKovDev/chat-bot/internal/infrastructure/nlp"
	"github.com/jackc/pgx/v5"
)

func TestReadinessProviderReportsReadyWhenRuntimeDependenciesAreBootstrapped(t *testing.T) {
	t.Parallel()

	nlp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ready","model":"deterministic","dimension":384}`))
	}))
	defer nlp.Close()

	provider := NewReadinessProvider(fakeReadinessDB{}, infranlp.EmbedderConfig{
		BaseURL:           nlp.URL,
		Timeout:           time.Second,
		ExpectedDimension: 384,
	})

	resp := provider(context.Background())
	if !resp.Ready {
		t.Fatalf("readiness should pass: %+v", resp.Checks)
	}
	for name, check := range resp.Checks {
		if !check.Ready {
			t.Fatalf("check %s should be ready: %+v", name, check)
		}
	}
}

func TestReadinessProviderFailsWhenSeedsAreMissing(t *testing.T) {
	t.Parallel()

	nlp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ready","model":"deterministic","dimension":384}`))
	}))
	defer nlp.Close()

	db := fakeReadinessDB{intents: 7, examples: 20, articles: 3, chunks: 3, accounts: 0, bookings: 4, workspaceBookings: 4, payments: 5}
	provider := NewReadinessProvider(db, infranlp.EmbedderConfig{
		BaseURL:           nlp.URL,
		Timeout:           time.Second,
		ExpectedDimension: 384,
	})

	resp := provider(context.Background())
	if resp.Ready {
		t.Fatalf("readiness should fail when seed data is missing: %+v", resp.Checks)
	}
	if resp.Checks["seed_data"].Ready {
		t.Fatalf("seed_data check should fail: %+v", resp.Checks["seed_data"])
	}
	if !strings.Contains(resp.Checks["seed_data"].Message, "demo accounts") {
		t.Fatalf("seed_data message should name missing demo accounts: %+v", resp.Checks["seed_data"])
	}
}

func TestReadinessProviderFailsWhenOperatorTablesAreMissing(t *testing.T) {
	t.Parallel()

	nlp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ready","model":"deterministic","dimension":384}`))
	}))
	defer nlp.Close()

	db := fakeReadinessDB{operatorQueueMissing: true}
	provider := NewReadinessProvider(db, infranlp.EmbedderConfig{
		BaseURL:           nlp.URL,
		Timeout:           time.Second,
		ExpectedDimension: 384,
	})

	resp := provider(context.Background())
	if resp.Ready {
		t.Fatalf("readiness should fail when operator tables are missing: %+v", resp.Checks)
	}
	if resp.Checks["operator_tables"].Ready {
		t.Fatalf("operator_tables check should fail: %+v", resp.Checks["operator_tables"])
	}
	if !strings.Contains(resp.Checks["operator_tables"].Message, "operator_queue") {
		t.Fatalf("operator_tables message should name missing table: %+v", resp.Checks["operator_tables"])
	}
}

type fakeReadinessDB struct {
	pingErr              error
	migration            int64
	vector               string
	operatorQueueMissing bool
	operators            int
	intents              int
	examples             int
	articles             int
	chunks               int
	accounts             int
	bookings             int
	workspaceBookings    int
	payments             int
}

func (f fakeReadinessDB) Ping(context.Context) error {
	return f.pingErr
}

func (f fakeReadinessDB) QueryRow(_ context.Context, query string, _ ...any) pgx.Row {
	if f.migration == 0 {
		f.migration = expectedMigrationVersion
	}
	if f.vector == "" {
		f.vector = "0.8.0"
	}
	if f.operators == 0 {
		f.operators = 2
	}
	if f.intents == 0 && f.examples == 0 && f.articles == 0 && f.chunks == 0 && f.accounts == 0 && f.bookings == 0 && f.workspaceBookings == 0 && f.payments == 0 {
		f.intents = 7
		f.examples = 20
		f.articles = 3
		f.chunks = 3
		f.accounts = 4
		f.bookings = 4
		f.workspaceBookings = 4
		f.payments = 5
	}

	switch {
	case strings.Contains(query, "goose_db_version"):
		return fakeRow{values: []any{f.migration}}
	case strings.Contains(query, "pg_extension"):
		return fakeRow{values: []any{f.vector}}
	case strings.Contains(query, "to_regclass('public.operators')"):
		return fakeRow{values: []any{true, !f.operatorQueueMissing, true, true}}
	case strings.Contains(query, "COUNT(*) FROM operators"):
		return fakeRow{values: []any{f.operators}}
	case strings.Contains(query, "FROM intents"):
		return fakeRow{values: []any{
			f.intents,
			f.examples,
			f.articles,
			f.chunks,
			f.accounts,
			f.bookings,
			f.workspaceBookings,
			f.payments,
		}}
	default:
		return fakeRow{err: errors.New("unexpected query")}
	}
}

type fakeRow struct {
	values []any
	err    error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for index, value := range r.values {
		switch target := dest[index].(type) {
		case *int64:
			*target = value.(int64)
		case *int:
			*target = value.(int)
		case *string:
			*target = value.(string)
		case *bool:
			*target = value.(bool)
		}
	}
	return nil
}
