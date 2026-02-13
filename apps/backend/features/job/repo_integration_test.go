package job_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"qurio/apps/backend/features/job"
	"qurio/apps/backend/features/source"
	"qurio/apps/backend/internal/testutils"
)

func TestJobRepo_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	s := testutils.NewIntegrationSuite(t)
	s.Setup()
	defer s.Teardown()

	jobRepo := job.NewPostgresRepo(s.DB)
	sourceRepo := source.NewPostgresRepo(s.DB)
	ctx := context.Background()

	// 1. Setup Source
	src := &source.Source{
		Type:        "web",
		URL:         "http://example.com",
		ContentHash: "hash-job-test",
		Name:        "Job Test Source",
	}
	err := sourceRepo.Save(ctx, src)
	require.NoError(t, err)

	// 2. Create Jobs
	j1 := &job.Job{
		SourceID: src.ID,
		Handler:  "test_handler",
		Payload:  json.RawMessage(`{"data": 1}`),
		Error:    "error 1",
	}
	err = jobRepo.Save(ctx, j1)
	require.NoError(t, err)

	// Sleep to ensure time difference for ordering test
	time.Sleep(100 * time.Millisecond)

	j2 := &job.Job{
		SourceID: src.ID,
		Handler:  "test_handler",
		Payload:  json.RawMessage(`{"data": 2}`),
		Error:    "error 2",
	}
	err = jobRepo.Save(ctx, j2)
	require.NoError(t, err)

	// 3. Verify List Ordering (DESC)
	jobs, err := jobRepo.List(ctx)
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, j2.ID, jobs[0].ID, "Newest job should be first")
	assert.Equal(t, j1.ID, jobs[1].ID, "Oldest job should be last")

	// 4. Verify Get
	gotJ1, err := jobRepo.Get(ctx, j1.ID)
	require.NoError(t, err)
	assert.Equal(t, j1.ID, gotJ1.ID)
	assert.Equal(t, "error 1", gotJ1.Error)

	// 5. Verify Delete
	err = jobRepo.Delete(ctx, j1.ID)
	require.NoError(t, err)

	// Verify it's gone
	_, err = jobRepo.Get(ctx, j1.ID)
	assert.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)

	count, err := jobRepo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// 6. Verify Cascade Delete (Hard Delete of Source)
	_, err = s.DB.ExecContext(ctx, "DELETE FROM sources WHERE id = $1", src.ID)
	require.NoError(t, err)

	// Now check if jobs are gone
	count, err = jobRepo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Jobs should be deleted via cascade when source is hard deleted")
}

func TestJobRepo_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	s := testutils.NewIntegrationSuite(t)
	s.Setup()
	defer s.Teardown()

	jobRepo := job.NewPostgresRepo(s.DB)
	ctx := context.Background()

	count, err := jobRepo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	list, err := jobRepo.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestJobRepo_Get_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	s := testutils.NewIntegrationSuite(t)
	s.Setup()
	defer s.Teardown()

	jobRepo := job.NewPostgresRepo(s.DB)
	ctx := context.Background()

	_, err := jobRepo.Get(ctx, uuid.NewString())
	assert.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
}
