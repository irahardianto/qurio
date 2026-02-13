package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"qurio/apps/backend/internal/config"
)

func TestLoadConfig(t *testing.T) {
	// Set env var directly to test envconfig logic
	os.Setenv("DB_HOST", "test-host")
	defer os.Unsetenv("DB_HOST")

	cfg, err := config.Load()
	assert.NoError(t, err)
	assert.Equal(t, "test-host", cfg.DBHost)
}

func TestLoadConfig_FromEnvFile(t *testing.T) {
	// Create a temp .env file
	content := []byte("DB_HOST=loaded-from-file")
	err := os.WriteFile(".env", content, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(".env")

	cfg, err := config.Load()
	assert.NoError(t, err)
	assert.Equal(t, "loaded-from-file", cfg.DBHost)
}

func TestLoadConfig_RerankAPIKey(t *testing.T) {
	os.Setenv("RERANK_API_KEY", "test-key")
	defer os.Unsetenv("RERANK_API_KEY")

	cfg, err := config.Load()
	assert.NoError(t, err)
	assert.Equal(t, "test-key", cfg.RerankAPIKey)
}

func TestLoadConfig_Toggles(t *testing.T) {
	os.Setenv("ENABLE_API", "false")
	os.Setenv("ENABLE_EMBEDDER_WORKER", "true")
	os.Setenv("INGESTION_CONCURRENCY", "10")
	defer os.Unsetenv("ENABLE_API")
	defer os.Unsetenv("ENABLE_EMBEDDER_WORKER")
	defer os.Unsetenv("INGESTION_CONCURRENCY")

	cfg, err := config.Load()
	assert.NoError(t, err)
	assert.False(t, cfg.EnableAPI)
	assert.True(t, cfg.EnableEmbedderWorker)
	assert.Equal(t, 10, cfg.IngestionConcurrency)
}
