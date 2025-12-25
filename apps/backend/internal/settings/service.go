package settings

import (
	"context"
)

type Settings struct {
	ID             int     `json:"-"`
	RerankProvider string  `json:"rerank_provider"`
	RerankAPIKey   string  `json:"rerank_api_key"`
	GeminiAPIKey   string  `json:"gemini_api_key"`
	SearchAlpha    float32 `json:"search_alpha"`
	SearchTopK     int     `json:"search_top_k"`
}

type Repository interface {
	Get(ctx context.Context) (*Settings, error)
	Update(ctx context.Context, s *Settings) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context) (*Settings, error) {
	return s.repo.Get(ctx)
}

func (s *Service) Update(ctx context.Context, set *Settings) error {
	return s.repo.Update(ctx, set)
}
