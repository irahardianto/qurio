package job_test

import (
	"context"
	"regexp"
	"testing"
	"time"
	"encoding/json"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"qurio/apps/backend/features/job"
)

func TestPostgresRepo_Save(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := job.NewPostgresRepo(db)

	t.Run("Success", func(t *testing.T) {
		j := &job.Job{
			SourceID: "src1",
			Handler:  "handler",
			Payload:  json.RawMessage(`{}`),
			Error:    "err",
		}

		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO failed_jobs")).
			WithArgs(j.SourceID, j.Handler, j.Payload, j.Error).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "retries"}).AddRow("1", time.Now(), 0))

		err := repo.Save(context.Background(), j)
		assert.NoError(t, err)
		assert.Equal(t, "1", j.ID)
	})
}

func TestPostgresRepo_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := job.NewPostgresRepo(db)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, source_id, handler, payload, error, retries, created_at FROM failed_jobs")).
			WillReturnRows(sqlmock.NewRows([]string{"id", "source_id", "handler", "payload", "error", "retries", "created_at"}).
				AddRow("1", "src1", "h", []byte(`{}`), "e", 0, time.Now()))

		jobs, err := repo.List(context.Background())
		assert.NoError(t, err)
		assert.Len(t, jobs, 1)
	})
}

func TestPostgresRepo_Get(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := job.NewPostgresRepo(db)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, source_id, handler, payload, error, retries, created_at FROM failed_jobs WHERE id = $1")).
			WithArgs("1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "source_id", "handler", "payload", "error", "retries", "created_at"}).
				AddRow("1", "src1", "h", []byte(`{}`), "e", 0, time.Now()))

		j, err := repo.Get(context.Background(), "1")
		assert.NoError(t, err)
		assert.Equal(t, "1", j.ID)
	})
}

func TestPostgresRepo_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := job.NewPostgresRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM failed_jobs WHERE id = $1")).
		WithArgs("1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Delete(context.Background(), "1")
	assert.NoError(t, err)
}

func TestPostgresRepo_Count(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := job.NewPostgresRepo(db)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM failed_jobs")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	count, err := repo.Count(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
}
