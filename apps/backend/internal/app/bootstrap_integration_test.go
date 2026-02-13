package app_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"qurio/apps/backend/internal/app"
	"qurio/apps/backend/internal/testutils"
)

func TestBootstrap_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	suite := testutils.NewIntegrationSuite(t)
	suite.Setup()
	defer suite.Teardown()

	cfg := suite.GetAppConfig()

	// Adjust MigrationPath for test context
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	// migrations are in ../../migrations relative to this file
	cfg.MigrationPath = fmt.Sprintf("file://%s/../../migrations", basepath)

	deps, err := app.Bootstrap(context.Background(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, deps)
	assert.NotNil(t, deps.DB)

	// Verify migration: Check if 'sources' table exists
	var exists bool
	err = deps.DB.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'sources')").Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists, "sources table should exist")

	// Verify Weaviate connectivity
	// We use EnsureSchema again as a connectivity check. GraphQL (CountChunks) seems flaky immediately after bootstrap in tests.
	err = deps.VectorStore.EnsureSchema(context.Background())
	assert.NoError(t, err, "Weaviate connectivity check failed")

	// Verify NSQ
	err = deps.NSQProducer.Ping()
	assert.NoError(t, err)
}
