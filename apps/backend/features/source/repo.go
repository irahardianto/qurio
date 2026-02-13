package source

import (
	"context"
	"database/sql"
	"time"

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
	query := `INSERT INTO sources (type, url, content_hash, max_depth, exclusions, name) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	return r.db.QueryRowContext(ctx, query, src.Type, src.URL, src.ContentHash, src.MaxDepth, pq.Array(src.Exclusions), src.Name).Scan(&src.ID)
}

func (r *PostgresRepo) UpdateStatus(ctx context.Context, id, status string) error {
	query := `UPDATE sources SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

func (r *PostgresRepo) List(ctx context.Context) ([]Source, error) {
	query := `SELECT id, type, url, status, max_depth, exclusions, name, updated_at FROM sources WHERE deleted_at IS NULL ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []Source
	for rows.Next() {
		var s Source
		if err := rows.Scan(&s.ID, &s.Type, &s.URL, &s.Status, &s.MaxDepth, pq.Array(&s.Exclusions), &s.Name, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return sources, nil
}

func (r *PostgresRepo) Get(ctx context.Context, id string) (*Source, error) {
	s := &Source{}
	query := `SELECT id, type, url, status, max_depth, exclusions, name, updated_at FROM sources WHERE id = $1 AND deleted_at IS NULL`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&s.ID, &s.Type, &s.URL, &s.Status, &s.MaxDepth, pq.Array(&s.Exclusions), &s.Name, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *PostgresRepo) SoftDelete(ctx context.Context, id string) error {
	query := `UPDATE sources SET deleted_at = NOW() WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *PostgresRepo) UpdateBodyHash(ctx context.Context, id, hash string) error {
	query := `UPDATE sources SET body_hash = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, hash, id)
	return err
}

func (r *PostgresRepo) Count(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM sources WHERE deleted_at IS NULL`
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

func (r *PostgresRepo) BulkCreatePages(ctx context.Context, pages []SourcePage) ([]string, error) {
	if len(pages) == 0 {
		return nil, nil
	}

	query := `INSERT INTO source_pages (source_id, url, status, depth) 
              VALUES ($1, $2, $3, $4) 
              ON CONFLICT (source_id, url) DO NOTHING
              RETURNING url`

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var newURLs []string
	for _, p := range pages {
		var u string
		err := stmt.QueryRowContext(ctx, p.SourceID, p.URL, p.Status, p.Depth).Scan(&u)
		if err == nil {
			newURLs = append(newURLs, u)
		} else if err != sql.ErrNoRows {
			// Real error
			return nil, err
		}
		// If ErrNoRows, it means conflict (duplicate), so we ignore
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return newURLs, nil
}

func (r *PostgresRepo) UpdatePageStatus(ctx context.Context, sourceID, url, status, errStr string) error {
	query := `UPDATE source_pages 
              SET status = $1, error = $2, updated_at = NOW() 
              WHERE source_id = $3 AND url = $4`
	_, err := r.db.ExecContext(ctx, query, status, errStr, sourceID, url)
	return err
}

func (r *PostgresRepo) GetPages(ctx context.Context, sourceID string) ([]SourcePage, error) {
	query := `SELECT id, source_id, url, status, depth, COALESCE(error, ''), created_at, updated_at 
              FROM source_pages 
              WHERE source_id = $1 
              ORDER BY created_at ASC`
	rows, err := r.db.QueryContext(ctx, query, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []SourcePage
	for rows.Next() {
		var p SourcePage
		if err := rows.Scan(&p.ID, &p.SourceID, &p.URL, &p.Status, &p.Depth, &p.Error, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, nil
}

func (r *PostgresRepo) DeletePages(ctx context.Context, sourceID string) error {
	query := `DELETE FROM source_pages WHERE source_id = $1`
	_, err := r.db.ExecContext(ctx, query, sourceID)
	return err
}

func (r *PostgresRepo) CountPendingPages(ctx context.Context, sourceID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM source_pages 
              WHERE source_id = $1 AND (status = 'pending' OR status = 'processing')`
	err := r.db.QueryRowContext(ctx, query, sourceID).Scan(&count)
	return count, err
}

func (r *PostgresRepo) ResetStuckPages(ctx context.Context, timeout time.Duration) (int64, error) {
	query := `UPDATE source_pages 
              SET status = 'pending', updated_at = NOW(), error = 'timeout_reset' 
              WHERE status = 'processing' AND updated_at < $1`

	cutoff := time.Now().Add(-timeout)

	result, err := r.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
