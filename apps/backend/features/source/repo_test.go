package source_test

import (
	"context"
	"regexp"
	"testing"
	//"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"qurio/apps/backend/features/source"
)

func TestPostgresRepo_Save(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := source.NewPostgresRepo(db)

	src := &source.Source{
		Type:        "web",
		URL:         "http://example.com",
		ContentHash: "hash123",
		MaxDepth:    2,
		Exclusions:  []string{"/admin"},
		Name:        "Example",
	}

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO sources (type, url, content_hash, max_depth, exclusions, name) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`)).
		WithArgs(src.Type, src.URL, src.ContentHash, src.MaxDepth, pq.Array(src.Exclusions), src.Name).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("new-id"))

	err = repo.Save(context.Background(), src)
	assert.NoError(t, err)
	assert.Equal(t, "new-id", src.ID)
}

func TestPostgresRepo_Get(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := source.NewPostgresRepo(db)

	rows := sqlmock.NewRows([]string{"id", "type", "url", "status", "max_depth", "exclusions", "name"}).
		AddRow("id1", "web", "http://example.com", "active", 2, "{}", "Test")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, type, url, status, max_depth, exclusions, name FROM sources WHERE id = $1 AND deleted_at IS NULL`)).
		WithArgs("id1").
		WillReturnRows(rows)

	s, err := repo.Get(context.Background(), "id1")
	assert.NoError(t, err)
	assert.Equal(t, "id1", s.ID)
	assert.Equal(t, "Test", s.Name)
}
