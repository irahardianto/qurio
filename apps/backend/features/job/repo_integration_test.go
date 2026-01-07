package job_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

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

	// 4. Verify Cascade Delete
	// Delete the source
	err = sourceRepo.SoftDelete(ctx, src.ID)
	// Soft delete might not trigger cascade if foreign key is on the table itself and doesn't check deleted_at.
	// Usually CASCADE is on DELETE action. Soft delete is an UPDATE set deleted_at.
	// So SoftDelete might NOT remove jobs if they are physically present.
	// But the Plan says "Verify failed_jobs records are deleted when parent source is deleted (Cascade)".
	// Does it mean Soft Delete of source should delete jobs? Or hard delete?
	// If the requirement is "Cascade Delete", strictly speaking that's Database FK behavior on DELETE.
	// If the application does Soft Delete, the FK cascade won't trigger unless we Hard Delete.
	// Let's check if there is a Hard Delete or if Soft Delete logic explicitly deletes jobs.
	
	// Let's check if there's a Hard Delete in Source Repo. No, only SoftDelete.
	// Maybe the requirement implies that when we clean up sources (Hard Delete), jobs go away.
	// Or maybe the integration test should simulate a hard delete to verify the FK constraint.
	
	// Let's try Hard Delete manually to verify FK constraint.
	_, err = s.DB.ExecContext(ctx, "DELETE FROM sources WHERE id = $1", src.ID)
	require.NoError(t, err)

	// Now check if jobs are gone
	count, err := jobRepo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Jobs should be deleted via cascade when source is hard deleted")
}
