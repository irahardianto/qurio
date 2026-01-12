package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nsqio/go-nsq"
	"qurio/apps/backend/internal/middleware"
)

type EmbedderConsumer struct {
	embedder Embedder
	store    VectorStore
}

func NewEmbedderConsumer(e Embedder, s VectorStore) *EmbedderConsumer {
	return &EmbedderConsumer{
		embedder: e,
		store:    s,
	}
}

func (h *EmbedderConsumer) HandleMessage(m *nsq.Message) error {
	if len(m.Body) == 0 {
		return nil
	}

	var payload IngestEmbedPayload
	if err := json.Unmarshal(m.Body, &payload); err != nil {
		// Poison Pill: Invalid JSON, don't retry
		slog.Error("poison pill: invalid json", "error", err)
		return nil
	}

	ctx := context.Background()
	if payload.CorrelationID != "" {
		ctx = middleware.WithCorrelationID(ctx, payload.CorrelationID)
	}

	// Reconstruct Contextual String
	// Format:
	// Title: <Page Title>
	// URL: <Page URL>
	// Type: <Content Type>
	// Author: <Author> (Optional)
	// Created: <Created At> (Optional)
	// ---
	// <Raw Chunk Content>
	contextualString := fmt.Sprintf("Title: %s\nSource: %s\nPath: %s\nURL: %s\nType: %s",
		payload.Title, payload.SourceName, payload.Path, payload.SourceURL, payload.ChunkType)

	if payload.Author != "" {
		contextualString += fmt.Sprintf("\nAuthor: %s", payload.Author)
	}
	if payload.CreatedAt != "" {
		contextualString += fmt.Sprintf("\nCreated: %s", payload.CreatedAt)
	}

	contextualString += fmt.Sprintf("\n---\n%s", payload.Content)

	// Embed with Timeout
	// Embedder interface usually takes context.
	embedCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	vector, err := h.embedder.Embed(embedCtx, contextualString)
	if err != nil {
		slog.ErrorContext(ctx, "embedding failed", "error", err, "source_id", payload.SourceID, "url", payload.SourceURL)
		return err // Retry
	}

	// Store Chunk
	chunk := Chunk{
		Content:    payload.Content,
		Vector:     vector,
		SourceID:   payload.SourceID,
		SourceURL:  payload.SourceURL,
		ChunkIndex: payload.ChunkIndex,
		Type:       payload.ChunkType,
		Language:   payload.Language,
		Title:      payload.Title,
		SourceName: payload.SourceName,
		Author:     payload.Author,
		CreatedAt:  payload.CreatedAt,
		PageCount:  payload.PageCount,
	}

	if err := h.store.StoreChunk(embedCtx, chunk); err != nil {
		slog.ErrorContext(ctx, "store chunk failed", "error", err, "source_id", payload.SourceID, "url", payload.SourceURL)
		return err // Retry
	}

	slog.InfoContext(ctx, "chunk stored successfully", "source_id", payload.SourceID, "chunk_index", payload.ChunkIndex)
	return nil
}
