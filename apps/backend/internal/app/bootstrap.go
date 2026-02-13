package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	wstore "qurio/apps/backend/internal/adapter/weaviate"
	"qurio/apps/backend/internal/config"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/nsqio/go-nsq"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
)

type Dependencies struct {
	DB          *sql.DB
	VectorStore VectorStore
	NSQProducer *nsq.Producer
}

func Bootstrap(ctx context.Context, cfg *config.Config) (*Dependencies, error) {
	// Database
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	// Retry loop
	retryDelay := time.Duration(cfg.BootstrapRetryDelaySeconds) * time.Second
	for i := 0; i < cfg.BootstrapRetryAttempts; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		slog.Warn("failed to ping db, retrying...", "attempt", i+1)
		time.Sleep(retryDelay)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	// Migrations
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("migration driver error: %w", err)
	}
	// Assuming migrations are in the current working directory "migrations"
	// However, usually path needs to be absolute or relative to where binary is run.
	// main.go was likely running from apps/backend.
	// We will stick to "file://migrations" as per plan and assume correct cwd.
	m, err := migrate.NewWithDatabaseInstance(cfg.MigrationPath, "postgres", driver)
	if err != nil {
		return nil, fmt.Errorf("migration instance error: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return nil, fmt.Errorf("migration up error: %w", err)
	}

	// Weaviate
	wCfg := weaviate.Config{Host: cfg.WeaviateHost, Scheme: cfg.WeaviateScheme}
	wClient, err := weaviate.NewClient(wCfg)
	if err != nil {
		return nil, fmt.Errorf("weaviate client error: %w", err)
	}
	vecStore := wstore.NewStore(wClient)

	// Ensure Schema Retry
	if err := EnsureSchemaWithRetry(ctx, vecStore, cfg.BootstrapRetryAttempts, retryDelay); err != nil {
		return nil, fmt.Errorf("weaviate schema error: %w", err)
	}

	// NSQ Producer
	nsqCfg := nsq.NewConfig()
	// nsqCfg.MaxMsgSize = cfg.NSQMaxMsgSize // Field undefined in go-nsq v1.1.0
	producer, err := nsq.NewProducer(cfg.NSQDHost, nsqCfg)
	if err != nil {
		return nil, fmt.Errorf("nsq producer error: %w", err)
	}

	// Topic pre-creation (Logic from main.go)
	createTopics(cfg.NSQDHTTP)

	return &Dependencies{
		DB:          db,
		VectorStore: vecStore,
		NSQProducer: producer,
	}, nil
}

func createTopics(nsqdHTTP string) {
	create := func(topic string) {
		url := fmt.Sprintf("http://%s/topic/create?topic=%s", nsqdHTTP, topic)
		resp, err := http.Post(url, "application/json", nil) // #nosec G107 -- URL is built from internal NSQ config, not user input
		if err != nil {
			slog.Warn("failed to create NSQ topic", "topic", topic, "error", err)
			return
		}
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.Warn("failed to close NSQ topic creation response body", "error", closeErr)
		}
	}

	go func() {
		time.Sleep(2 * time.Second)
		create(config.TopicIngestWeb)
		create(config.TopicIngestFile)
		create(config.TopicIngestResult)
		create(config.TopicIngestEmbed)
	}()
}

// EnsureSchemaWithRetry delegates schema check to a helper with retry logic.
func EnsureSchemaWithRetry(ctx context.Context, store VectorStore, attempts int, delay time.Duration) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = store.EnsureSchema(ctx); err == nil {
			return nil
		}
		if i < attempts-1 {
			time.Sleep(delay)
		}
	}
	return err
}
