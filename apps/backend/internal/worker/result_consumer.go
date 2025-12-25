package worker

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nsqio/go-nsq"
	"qurio/apps/backend/internal/text"
)

type ResultConsumer struct {
	embedder Embedder
	store    VectorStore
	updater  SourceStatusUpdater
}

func NewResultConsumer(e Embedder, s VectorStore, u SourceStatusUpdater) *ResultConsumer {
	return &ResultConsumer{embedder: e, store: s, updater: u}
}

func (h *ResultConsumer) HandleMessage(m *nsq.Message) error {
	if len(m.Body) == 0 {
		return nil
	}

	var payload struct {
		SourceID string `json:"source_id"`
		Content  string `json:"content"`
		URL      string `json:"url"`
		Status   string `json:"status,omitempty"` // "success" or "failed"
		Error    string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(m.Body, &payload); err != nil {
		slog.Error("invalid message format", "error", err)
		return nil // Don't retry invalid messages
	}

	ctx := context.Background()
	
	if payload.Status == "failed" {
		slog.Error("ingestion failed", "source_id", payload.SourceID, "error", payload.Error)
		if err := h.updater.UpdateStatus(ctx, payload.SourceID, "failed"); err != nil {
			slog.Warn("failed to update status to failed", "error", err)
		}
		return nil
	}

	slog.Info("received result", "source_id", payload.SourceID, "content_len", len(payload.Content))

	// 0. Delete Old Chunks (Idempotency)
	if payload.URL != "" {
		if err := h.store.DeleteChunksByURL(ctx, payload.SourceID, payload.URL); err != nil {
			slog.Error("failed to delete old chunks", "error", err, "source_id", payload.SourceID, "url", payload.URL)
			return err // Retry on error to ensure consistency
		}
	}

	// 1. Update Hash
	hash := sha256.Sum256([]byte(payload.Content))
	hashStr := fmt.Sprintf("%x", hash)
	if err := h.updater.UpdateBodyHash(ctx, payload.SourceID, hashStr); err != nil {
		slog.Warn("failed to update body hash", "error", err)
	}

	// 2. Chunk
	chunks := text.Chunk(payload.Content, 512, 50)
	if len(chunks) == 0 {
		slog.Warn("no chunks generated", "source_id", payload.SourceID)
		_ = h.updater.UpdateStatus(ctx, payload.SourceID, "completed")
		return nil
	}

	// 3. Embed & Store
	for i, c := range chunks {
		err := func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			vector, err := h.embedder.Embed(ctx, c)
			if err != nil {
				slog.Error("embed failed", "error", err)
				return err
			}

			chunk := Chunk{
				Content:    c,
				Vector:     vector,
				SourceID:   payload.SourceID,
				SourceURL:  payload.URL,
				ChunkIndex: i,
			}

			if err := h.store.StoreChunk(ctx, chunk); err != nil {
				slog.Error("store failed", "error", err)
				return err
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}

	slog.Info("stored chunks", "count", len(chunks), "source_id", payload.SourceID)
	
	// 4. Update Status
	if err := h.updater.UpdateStatus(ctx, payload.SourceID, "completed"); err != nil {
		slog.Warn("failed to update status", "error", err)
	}

	return nil
}
