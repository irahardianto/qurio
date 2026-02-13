package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"qurio/apps/backend/internal/testutils"
)

func TestSmoke_Startup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping smoke test in short mode")
	}

	// 1. Start Infrastructure
	suite := testutils.NewIntegrationSuite(t)
	// suite.SkipMigrations = true // Try running migrations in suite first
	suite.Setup()
	defer suite.Teardown()

	// 2. Configure App to use Infrastructure
	cfg := suite.GetAppConfig()
	cfg.EnableAPI = true // Ensure API is enabled for smoke test

	// Adjust MigrationPath
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	// migrations are in ./migrations relative to apps/backend root
	cfg.MigrationPath = fmt.Sprintf("file://%s/migrations", basepath)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// 3. Run App in Background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := run(ctx, cfg, logger)
		// Context canceled is expected on shutdown
		if err != nil && err != context.Canceled && err.Error() != "http: Server closed" {
			t.Logf("app run exited: %v", err)
		}
	}()

	// 4. Wait for Health Check
	require.Eventually(t, func() bool {
		resp, err := http.Get("http://localhost:8081/health")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 500*time.Millisecond)
}
