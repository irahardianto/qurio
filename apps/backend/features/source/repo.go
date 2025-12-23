package source

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
)

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *PostgresRepo {
	return &PostgresRepo{db: db}
}

func (r *PostgresRepo) ExistsByHash(ctx context.Context, hash string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM sources WHERE content_hash = $1 AND deleted_at IS NULL)`
	err := r.db.QueryRowContext(ctx, query, hash).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r *PostgresRepo) Save(ctx context.Context, src *Source) error {
	query := `INSERT INTO sources (type, url, content_hash, max_depth, exclusions) VALUES ($1, $2, $3, $4, $5) RETURNING id`
	return r.db.QueryRowContext(ctx, query, src.Type, src.URL, src.ContentHash, src.MaxDepth, pq.Array(src.Exclusions)).Scan(&src.ID)
}

func (r *PostgresRepo) UpdateStatus(ctx context.Context, id, status string) error {
	query := `UPDATE sources SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

func (r *PostgresRepo) List(ctx context.Context) ([]Source, error) {
	query := `SELECT id, type, url, status, max_depth, exclusions FROM sources WHERE deleted_at IS NULL ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []Source
	for rows.Next() {
		var s Source
		if err := rows.Scan(&s.ID, &s.Type, &s.URL, &s.Status, &s.MaxDepth, pq.Array(&s.Exclusions)); err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return sources, nil
}

func (r *PostgresRepo) Get(ctx context.Context, id string) (*Source, error) {
	s := &Source{}
	query := `SELECT id, type, url, status, max_depth, exclusions FROM sources WHERE id = $1 AND deleted_at IS NULL`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&s.ID, &s.Type, &s.URL, &s.Status, &s.MaxDepth, pq.Array(&s.Exclusions))
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *PostgresRepo) SoftDelete(ctx context.Context, id string) error {
	query := `UPDATE sources SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *PostgresRepo) UpdateBodyHash(ctx context.Context, id, hash string) error {
	query := `UPDATE sources SET body_hash = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, hash, id)
	return err
}