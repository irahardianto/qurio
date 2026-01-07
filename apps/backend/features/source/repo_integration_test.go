package source_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"qurio/apps/backend/features/source"
	"qurio/apps/backend/internal/testutils"
)

func TestSourceRepo_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	s := testutils.NewIntegrationSuite(t)
	s.Setup()
	defer s.Teardown()

	repo := source.NewPostgresRepo(s.DB)
	ctx := context.Background()

	// 1. Deduplication (Create)
	src := &source.Source{
		Type:        "web",
		URL:         "http://example.com",
		ContentHash: "hash1",
		MaxDepth:    1,
		Exclusions:  []string{},
		Name:        "Source 1",
	}
	err := repo.Save(ctx, src)
	require.NoError(t, err)
	assert.NotEmpty(t, src.ID)

	// Check ExistsByHash
	exists, err := repo.ExistsByHash(ctx, "hash1")
	require.NoError(t, err)
	assert.True(t, exists)

	// 2. Get and List
	retrieved, err := repo.Get(ctx, src.ID)
	require.NoError(t, err)
	assert.Equal(t, src.URL, retrieved.URL)

	list, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// 3. Update Status
	err = repo.UpdateStatus(ctx, src.ID, "completed")
	require.NoError(t, err)
	updated, err := repo.Get(ctx, src.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", updated.Status)

	// 4. Soft Delete
	err = repo.SoftDelete(ctx, src.ID)
	require.NoError(t, err)

	// Verify it's gone from standard Get/List
	_, err = repo.Get(ctx, src.ID)
	assert.Error(t, err) // Should be sql.ErrNoRows

	listAfterDelete, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, listAfterDelete, 0)

	// 5. Page Management
	pages := []source.SourcePage{
		{SourceID: src.ID, URL: "http://example.com/p1", Status: "pending", Depth: 0},
		{SourceID: src.ID, URL: "http://example.com/p2", Status: "processing", Depth: 1},
	}
	urls, err := repo.BulkCreatePages(ctx, pages)
	require.NoError(t, err)
	assert.Len(t, urls, 2)

	// CountPendingPages
	count, err := repo.CountPendingPages(ctx, src.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// ResetStuckPages
	resetCount, err := repo.ResetStuckPages(ctx, 1*time.Hour)
	require.NoError(t, err)
	assert.Equal(t, int64(0), resetCount)

	// UpdatePageStatus
	err = repo.UpdatePageStatus(ctx, src.ID, "http://example.com/p1", "completed", "")
	require.NoError(t, err)

	pagesRetrieved, err := repo.GetPages(ctx, src.ID)
	require.NoError(t, err)
	assert.Len(t, pagesRetrieved, 2)
	for _, p := range pagesRetrieved {
		if p.URL == "http://example.com/p1" {
			assert.Equal(t, "completed", p.Status)
		}
	}
}

func TestRepo_UniqueIndex_SoftDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	s := testutils.NewIntegrationSuite(t)
	s.Setup()
	defer s.Teardown()

	repo := source.NewPostgresRepo(s.DB)
	ctx := context.Background()

	// 1. Create Source A
	srcA := &source.Source{
		Type:        "web",
		URL:         "http://example.com/unique",
		ContentHash: "hash-unique",
		Name:        "Source Unique",
	}
	err := repo.Save(ctx, srcA)
	require.NoError(t, err)

	// 2. Soft Delete A
	err = repo.SoftDelete(ctx, srcA.ID)
	require.NoError(t, err)

	// 3. Create Source B (Same Hash) -> Should Succeed
	srcB := &source.Source{
		Type:        "web",
		URL:         "http://example.com/unique-2",
		ContentHash: "hash-unique",
		Name:        "Source Unique 2",
	}
	err = repo.Save(ctx, srcB)
	require.NoError(t, err)

	// 4. Create Source C (Same Hash) -> Should Fail (Active B exists)
	srcC := &source.Source{
		Type:        "web",
		URL:         "http://example.com/unique-3",
		ContentHash: "hash-unique",
		Name:        "Source Unique 3",
	}
	err = repo.Save(ctx, srcC)
	assert.Error(t, err)
}