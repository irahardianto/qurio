package retrieval

import (
	"context"
	"time"
	"qurio/apps/backend/internal/settings"
)

type SearchResult struct {
	Content  string                 `json:"content"`
	Score    float32                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
}

type SearchOptions struct {
	Alpha *float32
	Limit *int
}

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type VectorStore interface {
	Search(ctx context.Context, query string, vector []float32, alpha float32, limit int) ([]SearchResult, error)
}

type Reranker interface {
	Rerank(ctx context.Context, query string, docs []string) ([]int, error)
}

type Service struct {
	embedder Embedder
	store    VectorStore
	reranker Reranker
	settings *settings.Service
	logger   *QueryLogger
}

func NewService(e Embedder, s VectorStore, r Reranker, set *settings.Service, l *QueryLogger) *Service {
	return &Service{embedder: e, store: s, reranker: r, settings: set, logger: l}
}

func (s *Service) Search(ctx context.Context, query string, opts *SearchOptions) ([]SearchResult, error) {
	start := time.Now()
	var finalDocs []SearchResult
	var err error

	defer func() {
		if s.logger != nil && err == nil {
			s.logger.Log(QueryLogEntry{
				Query:      query,
				NumResults: len(finalDocs),
				Duration:   time.Since(start),
			})
		}
	}()

	// Get settings for defaults
	cfg, err := s.settings.Get(ctx)
	if err != nil {
		// Fallback defaults if settings fail (shouldn't happen)
		cfg = &settings.Settings{SearchAlpha: 0.5, SearchTopK: 10}
	}

	// Resolve params
	alpha := cfg.SearchAlpha
	if opts != nil && opts.Alpha != nil {
		alpha = *opts.Alpha
	}

	limit := cfg.SearchTopK
	if opts != nil && opts.Limit != nil {
		limit = *opts.Limit
	}

	// 1. Embed Query
	vec, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	// 2. Hybrid Search (BM25 + Vector)
	docs, err := s.store.Search(ctx, query, vec, alpha, limit)
	if err != nil {
		return nil, err
	}

	// 3. Rerank (if configured)
	if s.reranker != nil && len(docs) > 0 {
		// Extract content for reranker
		contents := make([]string, len(docs))
		for i, d := range docs {
			contents[i] = d.Content
		}

		indices, err := s.reranker.Rerank(ctx, query, contents)
		if err != nil {
			return nil, err
		}
		
		reranked := make([]SearchResult, len(indices))
		for i, idx := range indices {
			if idx < len(docs) {
				reranked[i] = docs[idx]
			}
		}
		finalDocs = reranked
		return reranked, nil
	}

	finalDocs = docs
	return docs, nil
}