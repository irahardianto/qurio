package app_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"qurio/apps/backend/internal/app"
	"qurio/apps/backend/internal/config"
	"qurio/apps/backend/internal/testutils"
)

func TestBootstrap_Resilience_DBDown(t *testing.T) {
	cfg := &config.Config{
		DBHost:                     "localhost",
		DBPort:                     54322, // Random port likely closed
		DBUser:                     "test",
		DBPass:                     "test",
		DBName:                     "test",
		BootstrapRetryAttempts:     1,
		BootstrapRetryDelaySeconds: 0,
	}

	start := time.Now()
	deps, err := app.Bootstrap(context.Background(), cfg)
	duration := time.Since(start)

	assert.Error(t, err)
	assert.Nil(t, deps)
	assert.Contains(t, err.Error(), "failed to ping db")
	// Since Open is lazy but we might fail DNS resolution or immediate dial if driver checks.
	// Actually sql.Open usually doesn't connect.
	// But Bootstrap calls db.Ping() immediately.
	// So it should fail on Ping.
	// We expect duration to be small since attempts=1, delay=0.
	assert.Less(t, duration, 2*time.Second)
}

func TestBootstrap_Resilience_WeaviateDown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Start DB via suite
	suite := testutils.NewIntegrationSuite(t)
	suite.Setup()
	defer suite.Teardown()

	goodCfg := suite.GetAppConfig()

	// Adjust MigrationPath
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	migrationPath := fmt.Sprintf("file://%s/../../migrations", basepath)

	// Config: Good DB, Bad Weaviate
	cfg := &config.Config{
		DBHost: goodCfg.DBHost,
		DBPort: goodCfg.DBPort,
		DBUser: goodCfg.DBUser,
		DBPass: goodCfg.DBPass,
		DBName: goodCfg.DBName,

		WeaviateHost:   "localhost:54322", // Bad host
		WeaviateScheme: "http",

		NSQDHost: goodCfg.NSQDHost, // Keep good NSQ to isolate Weaviate failure

		BootstrapRetryAttempts:     2,
		BootstrapRetryDelaySeconds: 1,
		MigrationPath:              migrationPath,
	}

	start := time.Now()
	deps, err := app.Bootstrap(context.Background(), cfg)
	duration := time.Since(start)

	assert.Error(t, err)
	assert.Nil(t, deps)
	// Expect Weaviate Schema Error
	// EnsureSchemaWithRetry is called.
	// It should fail after 2 attempts.
	assert.Contains(t, err.Error(), "weaviate schema error")
	assert.Greater(t, duration, 1*time.Second) // At least 1 delay
}
