package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"qurio/apps/backend/internal/app"
	"qurio/apps/backend/internal/config"
	"qurio/apps/backend/internal/logger"

	"github.com/nsqio/go-nsq"
)

func main() {
	// Initialize structured logger
	l := slog.New(logger.NewContextHandler(slog.NewJSONHandler(os.Stdout, nil)))
	slog.SetDefault(l)

	// 1. Load Config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Main context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx, cfg, l); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	// 2. Bootstrap Infrastructure (DB, Weaviate, NSQ Producer, Migrations)
	deps, err := app.Bootstrap(ctx, cfg)
	if err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}
	defer deps.DB.Close()

	// 3. Initialize App
	application, err := app.New(cfg, deps.DB, deps.VectorStore, deps.NSQProducer, logger, nil)
	if err != nil {
		return fmt.Errorf("failed to initialize app: %w", err)
	}

	// 4. Worker (Result Consumer) Setup
	nsqCfg := nsq.NewConfig()
	// nsqCfg.MaxMsgSize = cfg.NSQMaxMsgSize // Field undefined in go-nsq v1.1.0
	consumer, err := nsq.NewConsumer(config.TopicIngestResult, "backend", nsqCfg)
	if err != nil {
		slog.Error("failed to create NSQ consumer for results", "error", err)
	} else {
		// Use AddConcurrentHandlers
		consumer.AddConcurrentHandlers(nsq.HandlerFunc(func(m *nsq.Message) error {
			return application.ResultConsumer.HandleMessage(m)
		}), cfg.IngestionConcurrency)

		// Connect to Lookupd or NSQD
		if cfg.NSQLookupd != "" {
			if err := consumer.ConnectToNSQLookupd(cfg.NSQLookupd); err != nil {
				slog.Error("failed to connect to NSQLookupd", "error", err)
			} else {
				slog.Info("NSQ Result Consumer connected via Lookupd", "lookupd", cfg.NSQLookupd, "concurrency", cfg.IngestionConcurrency)
			}
		} else if cfg.NSQDHost != "" {
			if err := consumer.ConnectToNSQD(cfg.NSQDHost); err != nil {
				slog.Error("failed to connect to NSQD", "error", err)
			} else {
				slog.Info("NSQ Result Consumer connected via NSQD", "nsqd", cfg.NSQDHost, "concurrency", cfg.IngestionConcurrency)
			}
		}
	}

	// 5. Worker (Embedder Consumer) Setup
	if application.EmbedderConsumer != nil {
		consumer, err := nsq.NewConsumer(config.TopicIngestEmbed, "backend-embedder", nsqCfg)
		if err != nil {
			slog.Error("failed to create NSQ consumer for embed", "error", err)
		} else {
			consumer.AddConcurrentHandlers(nsq.HandlerFunc(func(m *nsq.Message) error {
				return application.EmbedderConsumer.HandleMessage(m)
			}), cfg.IngestionConcurrency)

			if cfg.NSQLookupd != "" {
				if err := consumer.ConnectToNSQLookupd(cfg.NSQLookupd); err != nil {
					slog.Error("failed to connect Embedder Consumer to NSQLookupd", "error", err)
				} else {
					slog.Info("NSQ Embedder Consumer connected via Lookupd", "lookupd", cfg.NSQLookupd)
				}
			} else if cfg.NSQDHost != "" {
				if err := consumer.ConnectToNSQD(cfg.NSQDHost); err != nil {
					slog.Error("failed to connect Embedder Consumer to NSQD", "error", err)
				} else {
					slog.Info("NSQ Embedder Consumer connected via NSQD", "nsqd", cfg.NSQDHost)
				}
			}
		}
	}

	// Background Janitor
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := application.SourceService.ResetStuckPages(context.Background()); err != nil {
					slog.Error("failed to reset stuck pages", "error", err)
				}
			}
		}
	}()

	// 5. Start Server
	if cfg.EnableAPI {
		if err := application.Run(ctx); err != nil {
			return fmt.Errorf("server failed: %w", err)
		}
	} else {
		slog.Info("API disabled, running in worker mode")
		<-ctx.Done()
	}
	return nil
}
