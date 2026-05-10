package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	appdecision "github.com/VladKovDev/chat-bot/internal/app/decision"
	apppresenter "github.com/VladKovDev/chat-bot/internal/app/presenter"
	appseed "github.com/VladKovDev/chat-bot/internal/app/seed"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository/postgres/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type SemanticCatalogRepository struct {
	querier *sqlc.Queries
}

func NewSemanticCatalogRepository(db sqlc.DBTX) *SemanticCatalogRepository {
	return &SemanticCatalogRepository{querier: sqlc.New(db)}
}

func (r *SemanticCatalogRepository) SearchIntentExamples(
	ctx context.Context,
	embedding []float64,
	locale string,
	limit int,
) ([]appdecision.IntentSearchResult, error) {
	if limit <= 0 {
		limit = 3
	}
	rows, err := r.querier.SearchIntentExamples(ctx, sqlc.SearchIntentExamplesParams{
		Embedding:   vectorLiteral(embedding),
		Locale:      firstNonEmptyString(locale, "ru"),
		ResultLimit: int32(limit),
	})
	if err != nil {
		return nil, err
	}

	results := make([]appdecision.IntentSearchResult, 0, len(rows))
	for _, row := range rows {
		results = append(results, appdecision.IntentSearchResult{
			IntentID:       pgUUIDToUUID(row.IntentID).String(),
			IntentKey:      row.IntentKey,
			Category:       row.Category,
			ResponseKey:    row.ResponseKey,
			Text:           row.Text,
			NormalizedText: row.NormalizedText,
			Locale:         row.Locale,
			Weight:         row.Weight,
			Confidence:     row.Confidence,
		})
	}
	return results, nil
}

func (r *SemanticCatalogRepository) SeedIntent(
	ctx context.Context,
	intent apppresenter.IntentDefinition,
) (uuid.UUID, error) {
	metadata, err := json.Marshal(map[string]any{
		"resolution_type":       intent.ResolutionType,
		"knowledge_key":         intent.KnowledgeKey,
		"action":                intent.Action,
		"fallback_response_key": intent.FallbackResponseKey,
		"result_response_keys":  intent.ResultResponseKeys,
		"e2e_coverage":          intent.E2ECoverage,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal intent metadata: %w", err)
	}

	dbIntent, err := r.querier.UpsertIntent(ctx, sqlc.UpsertIntentParams{
		Key:               intent.Key,
		Category:          intent.Category,
		ResponseKey:       intent.ResponseKey,
		RequiresAction:    strings.TrimSpace(intent.Action) != "",
		EscalateOnFailure: intent.ResolutionType == "operator_handoff",
		FallbackPolicy:    fallbackPolicyForIntent(intent),
		Active:            true,
		Metadata:          metadata,
	})
	if err != nil {
		return uuid.Nil, err
	}
	return pgUUIDToUUID(dbIntent.ID), nil
}

func (r *SemanticCatalogRepository) SeedIntentExample(
	ctx context.Context,
	intentID uuid.UUID,
	intentKey string,
	text string,
	embedding []float64,
) error {
	metadata, err := json.Marshal(map[string]any{
		"seed":       "intents.json",
		"intent_key": intentKey,
	})
	if err != nil {
		return fmt.Errorf("marshal intent example metadata: %w", err)
	}

	_, err = r.querier.UpsertIntentExample(ctx, sqlc.UpsertIntentExampleParams{
		IntentID:       uuidToPgUUID(intentID),
		Text:           text,
		NormalizedText: appdecision.NormalizeForSeed(text),
		Embedding:      vectorLiteral(embedding),
		Locale:         "ru",
		Weight:         1,
		Active:         true,
		Metadata:       metadata,
	})
	return err
}

func (r *SemanticCatalogRepository) SeedKnowledgeArticle(
	ctx context.Context,
	article appseed.KnowledgeArticle,
) (uuid.UUID, error) {
	metadata, err := json.Marshal(map[string]any{
		"fixture_id":  article.ID,
		"version":     article.Version,
		"updated_at":  article.UpdatedAt,
		"source_file": "knowledge-base.json",
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal knowledge article metadata: %w", err)
	}

	dbArticle, err := r.querier.UpsertKnowledgeArticle(ctx, sqlc.UpsertKnowledgeArticleParams{
		Key:      article.Key,
		Category: article.Category,
		Title:    article.Title,
		Body:     article.Content,
		Source:   firstNonEmptyString(article.Source, "seed"),
		Active:   true,
		Metadata: metadata,
	})
	if err != nil {
		return uuid.Nil, err
	}
	return pgUUIDToUUID(dbArticle.ID), nil
}

func (r *SemanticCatalogRepository) SeedKnowledgeChunk(
	ctx context.Context,
	articleID uuid.UUID,
	chunkIndex int,
	body string,
	embedding []float64,
) error {
	metadata, err := json.Marshal(map[string]any{
		"seed": "knowledge-base.json",
	})
	if err != nil {
		return fmt.Errorf("marshal knowledge chunk metadata: %w", err)
	}

	_, err = r.querier.UpsertKnowledgeChunk(ctx, sqlc.UpsertKnowledgeChunkParams{
		ArticleID:  uuidToPgUUID(articleID),
		ChunkIndex: int32(chunkIndex),
		Body:       body,
		Embedding:  vectorLiteral(embedding),
		Active:     true,
		Metadata:   metadata,
	})
	return err
}

func (r *SemanticCatalogRepository) SeedDemoData(ctx context.Context, dataset *appseed.Dataset) error {
	if dataset == nil {
		return fmt.Errorf("seed dataset is nil")
	}

	for _, fixture := range dataset.Users.Items {
		payload, err := marshalSeedPayload(fixture)
		if err != nil {
			return fmt.Errorf("marshal demo account %s: %w", fixture.ID, err)
		}
		if _, err := r.querier.UpsertDemoAccount(ctx, sqlc.UpsertDemoAccountParams{
			ID:          fixture.ID,
			UserID:      fixture.UserID,
			Identifiers: fixture.Identifiers,
			Email:       fixture.Email,
			Phone:       fixture.Phone,
			Status:      fixture.Status,
			Payload:     payload,
		}); err != nil {
			return fmt.Errorf("seed demo account %s: %w", fixture.ID, err)
		}
	}

	for _, fixture := range dataset.Bookings.Items {
		payload, err := marshalSeedPayload(fixture)
		if err != nil {
			return fmt.Errorf("marshal demo booking %s: %w", fixture.ID, err)
		}
		if _, err := r.querier.UpsertDemoBooking(ctx, sqlc.UpsertDemoBookingParams{
			ID:            fixture.ID,
			BookingNumber: fixture.BookingNumber,
			Identifiers:   fixture.Identifiers,
			Service:       fixture.Service,
			Master:        fixture.Master,
			BookingDate:   dateFromSeed(fixture.Date),
			BookingTime:   timeFromSeed(fixture.Time),
			Status:        fixture.Status,
			Payload:       payload,
		}); err != nil {
			return fmt.Errorf("seed demo booking %s: %w", fixture.ID, err)
		}
	}

	for _, fixture := range dataset.WorkspaceBookings.Items {
		payload, err := marshalSeedPayload(fixture)
		if err != nil {
			return fmt.Errorf("marshal demo workspace booking %s: %w", fixture.ID, err)
		}
		if _, err := r.querier.UpsertDemoWorkspaceBooking(ctx, sqlc.UpsertDemoWorkspaceBookingParams{
			ID:            fixture.ID,
			BookingNumber: fixture.BookingNumber,
			Identifiers:   fixture.Identifiers,
			WorkspaceType: fixture.WorkspaceType,
			BookingDate:   dateFromSeed(fixture.Date),
			BookingTime:   timeFromSeed(fixture.Time),
			DurationHours: int32PtrFromSeed(fixture.Duration),
			Status:        fixture.Status,
			Payload:       payload,
		}); err != nil {
			return fmt.Errorf("seed demo workspace booking %s: %w", fixture.ID, err)
		}
	}

	for _, fixture := range dataset.Payments.Items {
		payload, err := marshalSeedPayload(fixture)
		if err != nil {
			return fmt.Errorf("marshal demo payment %s: %w", fixture.ID, err)
		}
		if _, err := r.querier.UpsertDemoPayment(ctx, sqlc.UpsertDemoPaymentParams{
			ID:          fixture.ID,
			PaymentID:   fixture.PaymentID,
			Identifiers: fixture.Identifiers,
			AmountCents: int32(fixture.Amount),
			PaidAt:      timestamptzFromSeed(fixture.Date),
			Status:      fixture.Status,
			Purpose:     fixture.Purpose,
			Payload:     payload,
		}); err != nil {
			return fmt.Errorf("seed demo payment %s: %w", fixture.ID, err)
		}
	}

	return nil
}

func vectorLiteral(values []float64) string {
	var builder strings.Builder
	builder.WriteByte('[')
	for i, value := range values {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(strconv.FormatFloat(value, 'f', -1, 64))
	}
	builder.WriteByte(']')
	return builder.String()
}

func fallbackPolicyForIntent(intent apppresenter.IntentDefinition) string {
	switch intent.ResolutionType {
	case "operator_handoff":
		return "operator"
	case "knowledge":
		return "knowledge"
	case "business_lookup":
		if intent.Action == action.ActionEscalateToOperator {
			return "operator"
		}
		return "default"
	case "fallback":
		return "operator"
	default:
		return "default"
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func marshalSeedPayload(value any) ([]byte, error) {
	return json.Marshal(value)
}

func dateFromSeed(value string) pgtype.Date {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: parsed, Valid: true}
}

func timeFromSeed(value string) pgtype.Time {
	parsed, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		return pgtype.Time{}
	}
	microseconds := int64(parsed.Hour()*60*60+parsed.Minute()*60+parsed.Second()) * int64(time.Second/time.Microsecond)
	return pgtype.Time{Microseconds: microseconds, Valid: true}
}

func timestamptzFromSeed(value string) pgtype.Timestamptz {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: parsed, Valid: true}
}

func int32PtrFromSeed(value string) *int32 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
	if err != nil {
		return nil
	}
	result := int32(parsed)
	return &result
}

func nullablePgUUIDFromString(value string) pgtype.UUID {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return pgtype.UUID{}
	}
	return uuidToPgUUID(id)
}
