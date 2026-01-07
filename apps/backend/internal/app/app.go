package app

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"

	"qurio/apps/backend/features/job"
	"qurio/apps/backend/features/mcp"
	"qurio/apps/backend/features/source"
	"qurio/apps/backend/features/stats"
	"qurio/apps/backend/internal/adapter/gemini"
	"qurio/apps/backend/internal/adapter/reranker"
	"qurio/apps/backend/internal/config"
	"qurio/apps/backend/internal/middleware"
	"qurio/apps/backend/internal/retrieval"
	"qurio/apps/backend/internal/settings"
	"qurio/apps/backend/internal/worker"
)

type App struct {
	Handler        http.Handler
	SourceService  *source.Service
	ResultConsumer *worker.ResultConsumer
}

func New(
	cfg *config.Config,
	db Database,
	vecStore VectorStore,
	taskPub TaskPublisher,
	logger *slog.Logger,
) (*App, error) {
	
	// 5. Initialize Adapters & Services
	// vecStore is passed as interface

	// Feature: Settings
	// Cast db to *sql.DB for repositories that require it.
	// This allows us to use interfaces in the signature (for mocking with sqlmock)
	// while maintaining compatibility with existing repositories.
	sqlDB := db.(*sql.DB)

	settingsRepo := settings.NewPostgresRepo(sqlDB)
	settingsService := settings.NewService(settingsRepo)
	
	// Seed Gemini API Key from Config
	if cfg.GeminiAPIKey != "" {
		ctx := context.Background()
		set, err := settingsService.Get(ctx)
		if err == nil {
			// Update if empty
			if set.GeminiAPIKey == "" {
				set.GeminiAPIKey = cfg.GeminiAPIKey
				if err := settingsService.Update(ctx, set); err != nil {
					slog.Warn("failed to seed gemini api key", "error", err)
				} else {
					slog.Info("seeded gemini api key from environment")
				}
			}
		} else {
			slog.Warn("failed to fetch settings for seeding", "error", err)
		}
	}

	settingsHandler := settings.NewHandler(settingsService)

	// Feature: Source
	sourceRepo := source.NewPostgresRepo(sqlDB)
	sourceService := source.NewService(sourceRepo, taskPub, vecStore, settingsService)
	sourceHandler := source.NewHandler(sourceService)

	// Feature: Job
	jobRepo := job.NewPostgresRepo(sqlDB)
	jobService := job.NewService(jobRepo, taskPub, logger)
	jobHandler := job.NewHandler(jobService)

	// Feature: Stats
	statsHandler := stats.NewHandler(sourceRepo, jobRepo, vecStore)

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
	mux := http.NewServeMux()
	
	mux.Handle("POST /sources", middleware.CorrelationID(enableCORS(sourceHandler.Create)))
	mux.Handle("POST /sources/upload", middleware.CorrelationID(enableCORS(sourceHandler.Upload)))
	mux.Handle("GET /sources", middleware.CorrelationID(enableCORS(sourceHandler.List)))
	mux.Handle("GET /sources/{id}", middleware.CorrelationID(enableCORS(sourceHandler.Get)))
	mux.Handle("DELETE /sources/{id}", middleware.CorrelationID(enableCORS(sourceHandler.Delete)))
	mux.Handle("POST /sources/{id}/resync", middleware.CorrelationID(enableCORS(sourceHandler.ReSync)))
	mux.Handle("GET /sources/{id}/pages", middleware.CorrelationID(enableCORS(sourceHandler.GetPages)))

	mux.Handle("GET /settings", middleware.CorrelationID(enableCORS(settingsHandler.GetSettings)))
	mux.Handle("PUT /settings", middleware.CorrelationID(enableCORS(settingsHandler.UpdateSettings)))

	mux.Handle("GET /jobs/failed", middleware.CorrelationID(enableCORS(jobHandler.List)))
	mux.Handle("POST /jobs/{id}/retry", middleware.CorrelationID(enableCORS(jobHandler.Retry)))

	mux.Handle("GET /stats", middleware.CorrelationID(enableCORS(statsHandler.GetStats)))

	// Feature: Retrieval & MCP
	queryLogger, err := retrieval.NewFileQueryLogger("data/logs/query.log")
	if err != nil {
		slog.Warn("failed to create query logger, falling back to stdout", "error", err)
		queryLogger = retrieval.NewQueryLogger(os.Stdout)
	}

	retrievalService := retrieval.NewService(geminiEmbedder, vecStore, rerankerClient, settingsService, queryLogger)
	mcpHandler := mcp.NewHandler(retrievalService, sourceService)
	mux.Handle("/mcp", middleware.CorrelationID(mcpHandler)) // Legacy POST endpoint
	
	// New SSE Endpoints
	mux.Handle("GET /mcp/sse", middleware.CorrelationID(enableCORS(mcpHandler.HandleSSE)))
	mux.Handle("POST /mcp/messages", middleware.CorrelationID(enableCORS(mcpHandler.HandleMessage)))
	
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Worker (Result Consumer) Setup
	sfAdapter := &sourceFetcherAdapter{repo: sourceRepo, settings: settingsService}
	pmAdapter := &pageManagerAdapter{repo: sourceRepo}
	
	resultConsumer := worker.NewResultConsumer(geminiEmbedder, vecStore, sourceRepo, jobRepo, sfAdapter, pmAdapter, taskPub)

	return &App{
		Handler:        mux,
		SourceService:  sourceService,
		ResultConsumer: resultConsumer,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:    ":8081",
		Handler: a.Handler,
	}

	go func() {
		<-ctx.Done()
		slog.Info("shutting down server...")
		if err := srv.Shutdown(context.Background()); err != nil {
			slog.Error("server shutdown failed", "error", err)
		}
	}()

	slog.Info("server starting", "port", 8081)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Adapter for SourceFetcher in Worker
type sourceFetcherAdapter struct {
	repo     source.Repository
	settings source.SettingsService
}

func (a *sourceFetcherAdapter) GetSourceDetails(ctx context.Context, id string) (string, string, error) {
	s, err := a.repo.Get(ctx, id)
	if err != nil {
		return "", "", err
	}
	return s.Type, s.URL, nil
}

func (a *sourceFetcherAdapter) GetSourceConfig(ctx context.Context, id string) (int, []string, string, string, error) {
	s, err := a.repo.Get(ctx, id)
	if err != nil {
		return 0, nil, "", "", err
	}
	
	set, err := a.settings.Get(ctx)
	apiKey := ""
	if err == nil && set != nil {
		apiKey = set.GeminiAPIKey
	}
	
	return s.MaxDepth, s.Exclusions, apiKey, s.Name, nil
}

// Adapter for PageManager
type pageManagerAdapter struct {
	repo source.Repository
}

func (a *pageManagerAdapter) BulkCreatePages(ctx context.Context, pages []worker.PageDTO) ([]string, error) {
	// Convert worker.PageDTO to source.SourcePage
	var srcPages []source.SourcePage
	for _, p := range pages {
		srcPages = append(srcPages, source.SourcePage{
			SourceID: p.SourceID,
			URL:      p.URL,
			Status:   p.Status,
			Depth:    p.Depth,
		})
	}
	return a.repo.BulkCreatePages(ctx, srcPages)
}

func (a *pageManagerAdapter) UpdatePageStatus(ctx context.Context, sourceID, url, status, err string) error {
	return a.repo.UpdatePageStatus(ctx, sourceID, url, status, err)
}

func (a *pageManagerAdapter) CountPendingPages(ctx context.Context, sourceID string) (int, error) {
	return a.repo.CountPendingPages(ctx, sourceID)
}

