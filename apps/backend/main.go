package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"qurio/apps/backend/internal/app"
	"qurio/apps/backend/internal/config"
	"qurio/apps/backend/internal/logger"
	"qurio/apps/backend/internal/vector"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/nsqio/go-nsq"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
)

func main() {
	// Initialize structured logger
	logger := slog.New(logger.NewContextHandler(slog.NewJSONHandler(os.Stdout, nil)))
	slog.SetDefault(logger)

	// 1. Load Config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// 2. Database Connection
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName)
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		slog.Error("failed to open db connection", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Retry connection
	for i := 0; i < 10; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		slog.Warn("failed to ping db, retrying...", "attempt", i+1, "max_attempts", 10)
		time.Sleep(2 * time.Second)
	}

	if err := db.Ping(); err != nil {
		slog.Error("failed to ping db after retries", "error", err)
		os.Exit(1)
	}

	// 3. Run Migrations
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		slog.Error("failed to create migration driver", "error", err)
		os.Exit(1)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		slog.Error("failed to create migration instance", "error", err)
		os.Exit(1)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations applied successfully")

	// 4. Weaviate Connection & Schema
	wCfg := weaviate.Config{
		Host:   cfg.WeaviateHost,
		Scheme: cfg.WeaviateScheme,
	}
	wClient, err := weaviate.NewClient(wCfg)
	if err != nil {
		slog.Error("failed to create weaviate client", "error", err)
		os.Exit(1)
	}

	wAdapter := vector.NewWeaviateClientAdapter(wClient)
	
	// Retry Weaviate Schema Ensure
	for i := 0; i < 10; i++ {
		if err := vector.EnsureSchema(context.Background(), wAdapter); err == nil {
			slog.Info("weaviate schema ensured")
			break
		}
		slog.Warn("failed to ensure weaviate schema, retrying...", "attempt", i+1, "error", err)
		time.Sleep(2 * time.Second)
	}

	if err := vector.EnsureSchema(context.Background(), wAdapter); err != nil {
		slog.Error("failed to ensure weaviate schema after retries", "error", err)
		os.Exit(1)
	}

	// NSQ Producer
	nsqCfg := nsq.NewConfig()
	nsqProducer, err := nsq.NewProducer(cfg.NSQDHost, nsqCfg)
	if err != nil {
		slog.Error("failed to create NSQ producer", "error", err)
		os.Exit(1)
	}

	// Pre-create 'ingest' topic to avoid consumer startup errors
	nsqHttpURL := fmt.Sprintf("http://%s:4151/topic/create?topic=ingest.task", "nsqd")
	nsqResultURL := fmt.Sprintf("http://%s:4151/topic/create?topic=ingest.result", "nsqd")
	
	// If NSQDHost contains port, strip it. Usually "host:port"
	host, _, _ := net.SplitHostPort(cfg.NSQDHost)
	if host != "" {
		nsqHttpURL = fmt.Sprintf("http://%s:4151/topic/create?topic=ingest.task", host)
		nsqResultURL = fmt.Sprintf("http://%s:4151/topic/create?topic=ingest.result", host)
	}
	
	// Fire and forget topic creation
	go func() {
		// Wait for nsqd to be ready
		time.Sleep(2 * time.Second)
		// Create ingest.task
		resp, err := http.Post(nsqHttpURL, "application/json", nil)
		if err != nil {
			slog.Warn("failed to pre-create ingest.task topic", "error", err, "url", nsqHttpURL)
		} else {
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				slog.Info("ingest.task topic pre-created successfully")
			}
		}
		
		// Create ingest.result
		resp2, err := http.Post(nsqResultURL, "application/json", nil)
		if err != nil {
			slog.Warn("failed to pre-create ingest.result topic", "error", err, "url", nsqResultURL)
		} else {
			defer resp2.Body.Close()
			if resp2.StatusCode == 200 {
				slog.Info("ingest.result topic pre-created successfully")
			}
		}
	}()

	// 5. Initialize App
	application, err := app.New(cfg, db, wClient, nsqProducer, logger)
	if err != nil {
		slog.Error("failed to initialize app", "error", err)
		os.Exit(1)
	}

	// 6. Worker (Result Consumer) Setup
	nsqCfg = nsq.NewConfig()
	consumer, err := nsq.NewConsumer("ingest.result", "backend", nsqCfg)
	if err != nil {
		slog.Error("failed to create NSQ consumer for results", "error", err)
	} else {
		// Use AddConcurrentHandlers
		consumer.AddConcurrentHandlers(nsq.HandlerFunc(func(m *nsq.Message) error {
			return application.ResultConsumer.HandleMessage(m)
		}), cfg.IngestionConcurrency)
		
		// Connect to Lookupd
		if err := consumer.ConnectToNSQLookupd(cfg.NSQLookupd); err != nil {
			slog.Error("failed to connect to NSQLookupd", "error", err)
		} else {
			slog.Info("NSQ Result Consumer connected", "concurrency", cfg.IngestionConcurrency)
		}
	}

	// Background Janitor
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := application.SourceService.ResetStuckPages(context.Background()); err != nil {
					slog.Error("failed to reset stuck pages", "error", err)
				}
			}
		}
	}()

	// 7. Start Server
	slog.Info("server starting", "port", 8081)
	if err := http.ListenAndServe(":8081", application.Handler); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
