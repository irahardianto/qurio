package settings

import (
	"context"
	"database/sql"
)

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *PostgresRepo {
	return &PostgresRepo{db: db}
}

func (r *PostgresRepo) Get(ctx context.Context) (*Settings, error) {
	s := &Settings{}
	query := `SELECT id, rerank_provider, rerank_api_key, gemini_api_key, search_alpha, search_top_k FROM settings WHERE id = 1`
	err := r.db.QueryRowContext(ctx, query).Scan(&s.ID, &s.RerankProvider, &s.RerankAPIKey, &s.GeminiAPIKey, &s.SearchAlpha, &s.SearchTopK)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *PostgresRepo) Update(ctx context.Context, s *Settings) error {
	query := `
		UPDATE settings 
		SET rerank_provider = $1, rerank_api_key = $2, gemini_api_key = $3, search_alpha = $4, search_top_k = $5, updated_at = NOW()
		WHERE id = 1
	`
	_, err := r.db.ExecContext(ctx, query, s.RerankProvider, s.RerankAPIKey, s.GeminiAPIKey, s.SearchAlpha, s.SearchTopK)
	return err
}
