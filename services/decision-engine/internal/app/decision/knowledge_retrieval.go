package decision

import (
	"context"
	"fmt"
	"strings"

	apppresenter "github.com/VladKovDev/chat-bot/internal/app/presenter"
)

type KnowledgeSearchResult struct {
	ArticleKey string
	Category   string
	Title      string
	ChunkIndex int
	Body       string
	Confidence float64
}

type KnowledgeSearchRepository interface {
	SearchKnowledgeChunks(ctx context.Context, embedding []float64, limit int) ([]KnowledgeSearchResult, error)
}

type KnowledgeSearcher interface {
	Retrieve(ctx context.Context, text string, intent apppresenter.IntentDefinition) (*Candidate, error)
}

type KnowledgeRetrieverConfig struct {
	TopK int
}

type knowledgeRetriever struct {
	embedder Embedder
	repo     KnowledgeSearchRepository
	topK     int
}

func NewKnowledgeRetriever(
	embedder Embedder,
	repo KnowledgeSearchRepository,
	cfg KnowledgeRetrieverConfig,
) (KnowledgeSearcher, error) {
	if embedder == nil {
		return nil, fmt.Errorf("knowledge retriever embedder is required")
	}
	if repo == nil {
		return nil, fmt.Errorf("knowledge retriever repository is required")
	}
	topK := cfg.TopK
	if topK <= 0 {
		topK = 5
	}
	return &knowledgeRetriever{
		embedder: embedder,
		repo:     repo,
		topK:     topK,
	}, nil
}

func (r *knowledgeRetriever) Retrieve(
	ctx context.Context,
	text string,
	intent apppresenter.IntentDefinition,
) (*Candidate, error) {
	if intent.ResolutionType != "knowledge" {
		return nil, nil
	}

	embedding, err := r.embedder.Embed(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("embed knowledge query: %w", err)
	}

	rows, err := r.repo.SearchKnowledgeChunks(ctx, embedding, r.topK*4)
	if err != nil {
		return nil, fmt.Errorf("search knowledge chunks: %w", err)
	}

	best, ok := selectKnowledgeResult(rows, intent)
	if !ok {
		return nil, nil
	}

	return &Candidate{
		IntentKey:  intent.Key,
		Confidence: best.Confidence,
		Source:     CandidateSourceKnowledgeChunk,
		Text:       best.Body,
		Metadata: map[string]any{
			"article_key":   best.ArticleKey,
			"category":      best.Category,
			"title":         best.Title,
			"chunk_index":   best.ChunkIndex,
			"knowledge_key": intent.KnowledgeKey,
		},
	}, nil
}

func selectKnowledgeResult(rows []KnowledgeSearchResult, intent apppresenter.IntentDefinition) (KnowledgeSearchResult, bool) {
	for _, row := range rows {
		if strings.TrimSpace(intent.KnowledgeKey) != "" && row.ArticleKey == intent.KnowledgeKey {
			return row, true
		}
	}
	for _, row := range rows {
		if row.Category == intent.Category {
			return row, true
		}
	}
	return KnowledgeSearchResult{}, false
}
