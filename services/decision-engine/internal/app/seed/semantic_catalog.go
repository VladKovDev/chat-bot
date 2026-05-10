package seed

import (
	"context"
	"fmt"
	"strings"

	appdecision "github.com/VladKovDev/chat-bot/internal/app/decision"
	apppresenter "github.com/VladKovDev/chat-bot/internal/app/presenter"
	"github.com/google/uuid"
)

type SemanticCatalogRepository interface {
	SeedIntent(ctx context.Context, intent apppresenter.IntentDefinition) (uuid.UUID, error)
	SeedIntentExample(ctx context.Context, intentID uuid.UUID, intentKey string, text string, embedding []float64) error
	SeedKnowledgeArticle(ctx context.Context, article KnowledgeArticle) (uuid.UUID, error)
	SeedKnowledgeChunk(ctx context.Context, articleID uuid.UUID, chunkIndex int, body string, embedding []float64) error
}

func SeedSemanticCatalog(
	ctx context.Context,
	catalog *apppresenter.IntentCatalog,
	dataset *Dataset,
	repo SemanticCatalogRepository,
	embedder appdecision.Embedder,
) error {
	if catalog == nil {
		return fmt.Errorf("intent catalog is nil")
	}
	if dataset == nil {
		return fmt.Errorf("seed dataset is nil")
	}
	if repo == nil {
		return fmt.Errorf("semantic catalog repository is nil")
	}
	if embedder == nil {
		return fmt.Errorf("semantic catalog embedder is nil")
	}

	for _, intent := range catalog.Intents {
		intentID, err := repo.SeedIntent(ctx, intent)
		if err != nil {
			return fmt.Errorf("seed intent %s: %w", intent.Key, err)
		}
		for _, example := range intent.Examples {
			example = strings.TrimSpace(example)
			if example == "" {
				continue
			}
			embedding, err := embedder.Embed(ctx, example)
			if err != nil {
				return fmt.Errorf("embed intent example %s: %w", intent.Key, err)
			}
			if err := repo.SeedIntentExample(ctx, intentID, intent.Key, example, embedding); err != nil {
				return fmt.Errorf("seed intent example %s: %w", intent.Key, err)
			}
		}
	}

	for _, article := range dataset.KnowledgeBase.Articles {
		articleID, err := repo.SeedKnowledgeArticle(ctx, article)
		if err != nil {
			return fmt.Errorf("seed knowledge article %s: %w", article.Key, err)
		}
		body := strings.TrimSpace(article.Content)
		if body == "" {
			continue
		}
		embedding, err := embedder.Embed(ctx, article.Title+"\n"+body)
		if err != nil {
			return fmt.Errorf("embed knowledge article %s: %w", article.Key, err)
		}
		if err := repo.SeedKnowledgeChunk(ctx, articleID, 0, body, embedding); err != nil {
			return fmt.Errorf("seed knowledge chunk %s: %w", article.Key, err)
		}
	}

	return nil
}
