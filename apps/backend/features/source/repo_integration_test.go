package source_test

import (
	"context"
	"sync"
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
	assert.Error(t, err)

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

	// ResetStuckPages (Nothing old enough yet)
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

func TestRepo_ResetStuckPages_Effectiveness(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	s := testutils.NewIntegrationSuite(t)
	s.Setup()
	defer s.Teardown()

	repo := source.NewPostgresRepo(s.DB)
	ctx := context.Background()

	// Create Source
	src := &source.Source{Type: "web", URL: "http://example.com", ContentHash: "hash-stuck", Name: "S"}
	repo.Save(ctx, src)

	// Create Stuck Page (processing)
	repo.BulkCreatePages(ctx, []source.SourcePage{
		{SourceID: src.ID, URL: "http://example.com/stuck", Status: "processing", Depth: 0},
	})

	// Manually backdate updated_at
	_, err := s.DB.Exec("UPDATE source_pages SET updated_at = NOW() - INTERVAL '2 hours' WHERE url = $1", "http://example.com/stuck")
	require.NoError(t, err)

	// Reset
	count, err := repo.ResetStuckPages(ctx, 1*time.Hour)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Verify status is pending
	pages, _ := repo.GetPages(ctx, src.ID)
	assert.Equal(t, "pending", pages[0].Status)
	assert.Equal(t, "timeout_reset", pages[0].Error)
}

func TestRepo_DeletePages(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	s := testutils.NewIntegrationSuite(t)
	s.Setup()
	defer s.Teardown()

	repo := source.NewPostgresRepo(s.DB)
	ctx := context.Background()

	src := &source.Source{Type: "web", URL: "http://example.com", ContentHash: "hash-del", Name: "S"}
	repo.Save(ctx, src)

	repo.BulkCreatePages(ctx, []source.SourcePage{
		{SourceID: src.ID, URL: "http://example.com/1", Status: "pending"},
		{SourceID: src.ID, URL: "http://example.com/2", Status: "completed"},
	})

	err := repo.DeletePages(ctx, src.ID)
	require.NoError(t, err)

	pages, err := repo.GetPages(ctx, src.ID)
	require.NoError(t, err)
	assert.Empty(t, pages)
}

func TestRepo_Concurrent_Page_Creation(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	s := testutils.NewIntegrationSuite(t)
	s.Setup()
	defer s.Teardown()

	repo := source.NewPostgresRepo(s.DB)
	ctx := context.Background()
	src := &source.Source{Type: "web", URL: "http://example.com", ContentHash: "hash-conc", Name: "S"}
	repo.Save(ctx, src)

	var wg sync.WaitGroup
	// Try to create SAME page from multiple routines
	// ON CONFLICT DO NOTHING should prevent errors
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			repo.BulkCreatePages(ctx, []source.SourcePage{
				{SourceID: src.ID, URL: "http://example.com/shared", Status: "pending"},
			})
		}()
	}
	wg.Wait()

	pages, _ := repo.GetPages(ctx, src.ID)
	assert.Len(t, pages, 1) // Should only be 1
}
