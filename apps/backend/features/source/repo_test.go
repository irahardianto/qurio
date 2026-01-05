package source_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"qurio/apps/backend/features/source"
)

func TestPostgresRepo_ExistsByHash(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := source.NewPostgresRepo(db)

	t.Run("Exists", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM sources WHERE content_hash = $1 AND deleted_at IS NULL)")).
			WithArgs("hash123").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		exists, err := repo.ExistsByHash(context.Background(), "hash123")
		assert.NoError(t, err)
		assert.True(t, exists)
	})
}

func TestPostgresRepo_Save(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := source.NewPostgresRepo(db)

	t.Run("Success", func(t *testing.T) {
		src := &source.Source{
			Type:        "web",
			URL:         "http://example.com",
			ContentHash: "hash",
			MaxDepth:    2,
			Exclusions:  []string{},
			Name:        "Example",
		}

		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO sources (type, url, content_hash, max_depth, exclusions, name) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id")).
			WithArgs(src.Type, src.URL, src.ContentHash, src.MaxDepth, pq.Array(src.Exclusions), src.Name).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))

		err := repo.Save(context.Background(), src)
		assert.NoError(t, err)
		assert.Equal(t, "1", src.ID)
	})
}

func TestPostgresRepo_Get(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := source.NewPostgresRepo(db)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "type", "url", "status", "max_depth", "exclusions", "name", "updated_at"}).
			AddRow("1", "web", "http://example.com", "pending", 2, pq.Array([]string{}), "Example", time.Now())

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, url, status, max_depth, exclusions, name, updated_at FROM sources WHERE id = $1 AND deleted_at IS NULL")).
			WithArgs("1").
			WillReturnRows(rows)

		s, err := repo.Get(context.Background(), "1")
		assert.NoError(t, err)
		assert.Equal(t, "1", s.ID)
	})
}

func TestPostgresRepo_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := source.NewPostgresRepo(db)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "type", "url", "status", "max_depth", "exclusions", "name", "updated_at"}).
			AddRow("1", "website", "http://example.com", "pending", 2, pq.Array([]string{}), "Example", time.Now())

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, url, status, max_depth, exclusions, name, updated_at FROM sources WHERE deleted_at IS NULL ORDER BY created_at DESC")).
			WillReturnRows(rows)

		sources, err := repo.List(context.Background())
		assert.NoError(t, err)
		assert.Len(t, sources, 1)
	})
}

func TestPostgresRepo_BulkCreatePages(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := source.NewPostgresRepo(db)

	t.Run("Success", func(t *testing.T) {
		pages := []source.SourcePage{
			{SourceID: "src1", URL: "http://example.com/1", Status: "pending", Depth: 1},
		}

		mock.ExpectBegin()
		stmt := mock.ExpectPrepare(regexp.QuoteMeta("INSERT INTO source_pages"))
		stmt.ExpectQuery().
			WithArgs("src1", "http://example.com/1", "pending", 1).
			WillReturnRows(sqlmock.NewRows([]string{"url"}).AddRow("http://example.com/1"))
		mock.ExpectCommit()

		urls, err := repo.BulkCreatePages(context.Background(), pages)
		assert.NoError(t, err)
		assert.Len(t, urls, 1)
	})
}

func TestPostgresRepo_UpdateStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := source.NewPostgresRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE sources SET status = $1, updated_at = NOW() WHERE id = $2")).
		WithArgs("completed", "src1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.UpdateStatus(context.Background(), "src1", "completed")
	assert.NoError(t, err)
}

func TestPostgresRepo_SoftDelete(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := source.NewPostgresRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE sources SET deleted_at = NOW() WHERE id = $1")).
		WithArgs("src1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.SoftDelete(context.Background(), "src1")
	assert.NoError(t, err)
}

func TestPostgresRepo_UpdateBodyHash(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := source.NewPostgresRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE sources SET body_hash = $1, updated_at = NOW() WHERE id = $2")).
		WithArgs("hash", "src1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.UpdateBodyHash(context.Background(), "src1", "hash")
	assert.NoError(t, err)
}

func TestPostgresRepo_Count(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := source.NewPostgresRepo(db)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM sources WHERE deleted_at IS NULL")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	count, err := repo.Count(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestPostgresRepo_UpdatePageStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := source.NewPostgresRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE source_pages SET status = $1, error = $2, updated_at = NOW() WHERE source_id = $3 AND url = $4")).
		WithArgs("failed", "err", "src1", "http://u.rl").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.UpdatePageStatus(context.Background(), "src1", "http://u.rl", "failed", "err")
	assert.NoError(t, err)
}

func TestPostgresRepo_GetPages(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := source.NewPostgresRepo(db)

	rows := sqlmock.NewRows([]string{"id", "source_id", "url", "status", "depth", "error", "created_at", "updated_at"}).
		AddRow("p1", "src1", "http://u.rl", "pending", 0, "", time.Now(), time.Now())

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, source_id, url, status, depth, COALESCE(error, ''), created_at, updated_at FROM source_pages")).
		WithArgs("src1").
		WillReturnRows(rows)

	pages, err := repo.GetPages(context.Background(), "src1")
	assert.NoError(t, err)
	assert.Len(t, pages, 1)
}

func TestPostgresRepo_DeletePages(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := source.NewPostgresRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM source_pages WHERE source_id = $1")).
		WithArgs("src1").
		WillReturnResult(sqlmock.NewResult(10, 10))

	err = repo.DeletePages(context.Background(), "src1")
	assert.NoError(t, err)
}

func TestPostgresRepo_CountPendingPages(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := source.NewPostgresRepo(db)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM source_pages WHERE source_id = $1 AND (status = 'pending' OR status = 'processing')")).
		WithArgs("src1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	count, err := repo.CountPendingPages(context.Background(), "src1")
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestPostgresRepo_ResetStuckPages(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := source.NewPostgresRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE source_pages SET status = 'pending', updated_at = NOW(), error = 'timeout_reset' WHERE status = 'processing' AND updated_at < $1")).
		WillReturnResult(sqlmock.NewResult(5, 5))

	affected, err := repo.ResetStuckPages(context.Background(), time.Minute)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), affected)
}
