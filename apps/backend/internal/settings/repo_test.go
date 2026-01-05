package settings_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"qurio/apps/backend/internal/settings"
)

func TestPostgresRepo_Get(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := settings.NewPostgresRepo(db)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "rerank_provider", "rerank_api_key", "gemini_api_key", "search_alpha", "search_top_k"}).
			AddRow(1, "cohere", "key1", "key2", 0.5, 10)

		// Regex matching for the query
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, rerank_provider, rerank_api_key, gemini_api_key, search_alpha, search_top_k FROM settings WHERE id = 1")).
			WillReturnRows(rows)

		s, err := repo.Get(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, s)
		assert.Equal(t, "cohere", s.RerankProvider)
		assert.Equal(t, float32(0.5), s.SearchAlpha)
	})

	t.Run("Error", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id")).
			WillReturnError(sqlmock.ErrCancelled)

		s, err := repo.Get(context.Background())
		assert.Error(t, err)
		assert.Nil(t, s)
	})
}

func TestPostgresRepo_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := settings.NewPostgresRepo(db)

	t.Run("Success", func(t *testing.T) {
		s := &settings.Settings{
			RerankProvider: "jina",
			RerankAPIKey:   "k1",
			GeminiAPIKey:   "k2",
			SearchAlpha:    0.7,
			SearchTopK:     20,
		}

		mock.ExpectExec(regexp.QuoteMeta("UPDATE settings SET rerank_provider = $1, rerank_api_key = $2, gemini_api_key = $3, search_alpha = $4, search_top_k = $5, updated_at = NOW() WHERE id = 1")).
			WithArgs(s.RerankProvider, s.RerankAPIKey, s.GeminiAPIKey, s.SearchAlpha, s.SearchTopK).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Update(context.Background(), s)
		assert.NoError(t, err)
	})
}
