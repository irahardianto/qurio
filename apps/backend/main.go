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

	"qurio/apps/backend/features/mcp"
	"qurio/apps/backend/features/source"
	"qurio/apps/backend/internal/adapter/gemini"
	"qurio/apps/backend/internal/adapter/reranker"
	wstore "qurio/apps/backend/internal/adapter/weaviate"
	"qurio/apps/backend/internal/config"
	"qurio/apps/backend/internal/retrieval"
	"qurio/apps/backend/internal/vector"
	"qurio/apps/backend/internal/settings"
	"qurio/apps/backend/internal/worker"
	"qurio/apps/backend/internal/middleware"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/nsqio/go-nsq"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
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

	// 5. Initialize Adapters & Services
	vecStore := wstore.NewStore(wClient)

	// NSQ Producer
	nsqCfg := nsq.NewConfig()
	nsqProducer, err := nsq.NewProducer(cfg.NSQDHost, nsqCfg)
	if err != nil {
		slog.Error("failed to create NSQ producer", "error", err)
		os.Exit(1)
	}

	// Pre-create 'ingest' topic to avoid consumer startup errors
	// NSQ creates topics lazily on publish, but consumers querying lookupd will fail 404 until then.
	// We hit the nsqd http api to create it explicitly.
	// cfg.NSQDHost is "nsqd:4150" (TCP), we need HTTP port 4151
	// Assuming nsqd host is resolvable and port 4151 is standard
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

	// Feature: Settings
	settingsRepo := settings.NewPostgresRepo(db)
	settingsService := settings.NewService(settingsRepo)
	settingsHandler := settings.NewHandler(settingsService)

	// Feature: Source
	sourceRepo := source.NewPostgresRepo(db)
	sourceService := source.NewService(sourceRepo, nsqProducer, vecStore, settingsService)
	sourceHandler := source.NewHandler(sourceService)

	// Adapters: Dynamic
	geminiEmbedder := gemini.NewDynamicEmbedder(settingsService)
	rerankerClient := reranker.NewDynamicClient(settingsService)

	// Middleware: CORS
	enableCORS := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next(w, r)
		}
	}

	// Routes
	http.Handle("POST /sources", middleware.CorrelationID(enableCORS(sourceHandler.Create)))
	http.Handle("GET /sources", middleware.CorrelationID(enableCORS(sourceHandler.List)))
	http.Handle("GET /sources/{id}", middleware.CorrelationID(enableCORS(sourceHandler.Get)))
	http.Handle("DELETE /sources/{id}", middleware.CorrelationID(enableCORS(sourceHandler.Delete)))
	http.Handle("POST /sources/{id}/resync", middleware.CorrelationID(enableCORS(sourceHandler.ReSync)))

	http.Handle("GET /settings", middleware.CorrelationID(enableCORS(settingsHandler.GetSettings)))
	http.Handle("PUT /settings", middleware.CorrelationID(enableCORS(settingsHandler.UpdateSettings)))

	// Feature: Retrieval & MCP
	queryLogger, err := retrieval.NewFileQueryLogger("data/logs/query.log")
	if err != nil {
		slog.Warn("failed to create query logger, falling back to stdout", "error", err)
		queryLogger = retrieval.NewQueryLogger(os.Stdout)
	}

	retrievalService := retrieval.NewService(geminiEmbedder, vecStore, rerankerClient, queryLogger)
	mcpHandler := mcp.NewHandler(retrievalService)
	http.Handle("/mcp", middleware.CorrelationID(mcpHandler)) // Legacy POST endpoint
	
	// New SSE Endpoints
	http.Handle("GET /mcp/sse", middleware.CorrelationID(enableCORS(mcpHandler.HandleSSE)))
	http.Handle("POST /mcp/messages", middleware.CorrelationID(enableCORS(mcpHandler.HandleMessage)))

	// Worker (Result Consumer)
	resultConsumer := worker.NewResultConsumer(geminiEmbedder, vecStore, sourceRepo)
	
	nsqCfg = nsq.NewConfig()
	// Consume 'ingest.result' topic, 'worker' channel (or 'backend' channel to be distinct from python worker channel if relevant)
	// Python worker consumes 'ingest.task'.
	// Go consumes 'ingest.result'.
	consumer, err := nsq.NewConsumer("ingest.result", "backend", nsqCfg)
	if err != nil {
		slog.Error("failed to create NSQ consumer for results", "error", err)
	} else {
		consumer.AddHandler(nsq.HandlerFunc(func(m *nsq.Message) error {
			return resultConsumer.HandleMessage(m)
		}))
		// Connect to Lookupd
		if err := consumer.ConnectToNSQLookupd(cfg.NSQLookupd); err != nil {
			slog.Error("failed to connect to NSQLookupd", "error", err)
		} else {
			slog.Info("NSQ Result Consumer connected")
		}
	}

	// 6. Start Server
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	slog.Info("server starting", "port", 8081)
	if err := http.ListenAndServe(":8081", nil); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}