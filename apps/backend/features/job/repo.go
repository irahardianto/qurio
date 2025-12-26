package job

import (
	"context"
	"database/sql"
	"encoding/json"
)

type Repository interface {
	Save(ctx context.Context, job *Job) error
	List(ctx context.Context) ([]Job, error)
	Get(ctx context.Context, id string) (*Job, error)
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int, error)
}

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *PostgresRepo {
	return &PostgresRepo{db: db}
}

func (r *PostgresRepo) Save(ctx context.Context, job *Job) error {
	query := `INSERT INTO failed_jobs (source_id, handler, payload, error) VALUES ($1, $2, $3, $4) RETURNING id, created_at, retries`
	return r.db.QueryRowContext(ctx, query, job.SourceID, job.Handler, job.Payload, job.Error).Scan(&job.ID, &job.CreatedAt, &job.Retries)
}

func (r *PostgresRepo) List(ctx context.Context) ([]Job, error) {
	query := `SELECT id, source_id, handler, payload, error, retries, created_at FROM failed_jobs ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var j Job
		var payload []byte
		if err := rows.Scan(&j.ID, &j.SourceID, &j.Handler, &payload, &j.Error, &j.Retries, &j.CreatedAt); err != nil {
			return nil, err
		}
		j.Payload = json.RawMessage(payload)
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func (r *PostgresRepo) Get(ctx context.Context, id string) (*Job, error) {
	j := &Job{}
	var payload []byte
	query := `SELECT id, source_id, handler, payload, error, retries, created_at FROM failed_jobs WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&j.ID, &j.SourceID, &j.Handler, &payload, &j.Error, &j.Retries, &j.CreatedAt)
	if err != nil {
		return nil, err
	}
	j.Payload = json.RawMessage(payload)
	return j, nil
}

func (r *PostgresRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM failed_jobs WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *PostgresRepo) Count(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM failed_jobs`
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}
