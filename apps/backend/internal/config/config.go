package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

var ErrMissingRequired = errors.New("missing required configuration")

type Config struct {
	DBHost string `envconfig:"DB_HOST" default:"postgres"`
	DBPort int    `envconfig:"DB_PORT" default:"5432"`
	DBUser string `envconfig:"DB_USER" default:"qurio"`
	DBPass string `envconfig:"DB_PASS" default:"password"`
	DBName string `envconfig:"DB_NAME" default:"qurio"`

	WeaviateHost   string `envconfig:"WEAVIATE_HOST" default:"localhost:8080"`
	WeaviateScheme string `envconfig:"WEAVIATE_SCHEME" default:"http"`

	DoclingURL string `envconfig:"DOCLING_URL" default:"http://docling:8000"`
	NSQLookupd string `envconfig:"NSQ_LOOKUPD" default:"nsqlookupd:4161"`
	NSQDHost   string `envconfig:"NSQD_HOST" default:"nsqd:4150"`
	NSQDHTTP   string `envconfig:"NSQD_HTTP" default:"nsqd:4151"`

	EnableAPI            bool   `envconfig:"ENABLE_API" default:"true"`
	EnableEmbedderWorker bool   `envconfig:"ENABLE_EMBEDDER_WORKER" default:"false"`
	IngestionConcurrency int    `envconfig:"INGESTION_CONCURRENCY" default:"50"`
	MigrationPath        string `envconfig:"MIGRATION_PATH" default:"file://migrations"`
	GeminiAPIKey         string `envconfig:"GEMINI_API_KEY"`
	RerankAPIKey         string `envconfig:"RERANK_API_KEY"`
	NSQMaxMsgSize        int64  `envconfig:"NSQ_MAX_MSG_SIZE" default:"10485760"` // 10MB

	// Server
	ServerPort      int    `envconfig:"SERVER_PORT" default:"8081"`
	QueryLogPath    string `envconfig:"QUERY_LOG_PATH" default:"data/logs/query.log"`
	MaxUploadSizeMB int64  `envconfig:"MAX_UPLOAD_SIZE_MB" default:"50"`
	UploadDir       string `envconfig:"QURIO_UPLOAD_DIR" default:"./uploads"`

	// Resilience
	BootstrapRetryAttempts     int `envconfig:"BOOTSTRAP_RETRY_ATTEMPTS" default:"10"`
	BootstrapRetryDelaySeconds int `envconfig:"BOOTSTRAP_RETRY_DELAY_SECONDS" default:"2"`
}

func Load() (*Config, error) {
	// Try loading .env from current dir and repo root
	// Ignore errors, as env vars might be set in the shell
	_ = godotenv.Load(".env")

	// Try finding root .env (assuming 2 levels up if in apps/backend)
	cwd, _ := os.Getwd()
	rootEnv := filepath.Join(cwd, "../../.env")
	_ = godotenv.Load(rootEnv)

	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.DBHost == "" {
		return fmt.Errorf("%w: DB_HOST", ErrMissingRequired)
	}
	if c.DBUser == "" {
		return fmt.Errorf("%w: DB_USER", ErrMissingRequired)
	}
	if c.DBName == "" {
		return fmt.Errorf("%w: DB_NAME", ErrMissingRequired)
	}
	return nil
}
