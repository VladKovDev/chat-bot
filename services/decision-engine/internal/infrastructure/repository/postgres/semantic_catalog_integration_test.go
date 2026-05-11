package postgres

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	apppresenter "github.com/VladKovDev/chat-bot/internal/app/presenter"
	appseed "github.com/VladKovDev/chat-bot/internal/app/seed"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSemanticCatalogRepositorySearchIntentExamplesWithPgvector(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping pgvector integration test in short mode")
	}
	dsn := os.Getenv("PGVECTOR_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("PGVECTOR_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	repo := NewSemanticCatalogRepositoryWithDimension(pool, appseed.SemanticEmbeddingDimension)
	keys := []string{"test_semantic_payment", "test_semantic_booking"}
	for _, key := range keys {
		if _, err := pool.Exec(ctx, `DELETE FROM intents WHERE key = $1`, key); err != nil {
			t.Fatalf("delete stale intent %s: %v", key, err)
		}
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		for _, key := range keys {
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM intents WHERE key = $1`, key)
		}
	})

	paymentID, err := repo.SeedIntent(ctx, apppresenter.IntentDefinition{
		Key:            "test_semantic_payment",
		Category:       "payment",
		ResolutionType: "business_lookup",
		ResponseKey:    "payment_request_identifier",
		Action:         "find_payment",
	})
	if err != nil {
		t.Fatalf("seed payment intent: %v", err)
	}
	bookingID, err := repo.SeedIntent(ctx, apppresenter.IntentDefinition{
		Key:            "test_semantic_booking",
		Category:       "booking",
		ResolutionType: "business_lookup",
		ResponseKey:    "booking_request_identifier",
		Action:         "find_booking",
	})
	if err != nil {
		t.Fatalf("seed booking intent: %v", err)
	}

	if err := repo.SeedIntentExample(ctx, paymentID, "test_semantic_payment", "проверь оплату", basisVector(0, appseed.SemanticEmbeddingDimension)); err != nil {
		t.Fatalf("seed payment example: %v", err)
	}
	if err := repo.SeedIntentExample(ctx, bookingID, "test_semantic_booking", "проверь запись", basisVector(1, appseed.SemanticEmbeddingDimension)); err != nil {
		t.Fatalf("seed booking example: %v", err)
	}

	rows, err := repo.SearchIntentExamples(ctx, basisVector(0, appseed.SemanticEmbeddingDimension), "ru", 2)
	if err != nil {
		t.Fatalf("search intent examples: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("row count = %d, want 2", len(rows))
	}
	if rows[0].IntentKey != "test_semantic_payment" {
		t.Fatalf("top intent = %q, want test_semantic_payment; rows=%#v", rows[0].IntentKey, rows)
	}
	if rows[0].Confidence < 0.99 {
		t.Fatalf("top confidence = %.4f, want near 1", rows[0].Confidence)
	}
	if rows[1].IntentKey != "test_semantic_booking" {
		t.Fatalf("second intent = %q, want test_semantic_booking", rows[1].IntentKey)
	}
}

func TestSemanticCatalogRepositoryRejectsEmbeddingDimensionMismatch(t *testing.T) {
	t.Parallel()

	repo := NewSemanticCatalogRepositoryWithDimension(nil, appseed.SemanticEmbeddingDimension)
	err := repo.SeedKnowledgeChunk(context.Background(), [16]byte{}, 0, "body", basisVector(0, appseed.SemanticEmbeddingDimension-1))
	if err == nil {
		t.Fatal("expected dimension mismatch error")
	}
	if !strings.Contains(err.Error(), "embedding dimension") {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = repo.SearchIntentExamples(context.Background(), basisVector(0, appseed.SemanticEmbeddingDimension-1), "ru", 1)
	if err == nil {
		t.Fatal("expected search dimension mismatch error")
	}
	if !strings.Contains(err.Error(), "embedding dimension") {
		t.Fatalf("unexpected search error: %v", err)
	}
}

func basisVector(index int, dimension int) []float64 {
	values := make([]float64, dimension)
	values[index] = 1
	return values
}
