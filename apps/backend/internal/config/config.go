package config

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

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
	
	EnableAPI            bool `envconfig:"ENABLE_API" default:"true"`
	EnableEmbedderWorker bool `envconfig:"ENABLE_EMBEDDER_WORKER" default:"false"`
	IngestionConcurrency int  `envconfig:"INGESTION_CONCURRENCY" default:"50"`
	MigrationPath string `envconfig:"MIGRATION_PATH" default:"file://migrations"`
	GeminiAPIKey string `envconfig:"GEMINI_API_KEY"`
	RerankAPIKey string `envconfig:"RERANK_API_KEY"`
	NSQMaxMsgSize int64 `envconfig:"NSQ_MAX_MSG_SIZE" default:"10485760"` // 10MB
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
	return &cfg, err
}
