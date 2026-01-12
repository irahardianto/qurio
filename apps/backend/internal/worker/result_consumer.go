package worker

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/google/uuid"
	"github.com/nsqio/go-nsq"
	"qurio/apps/backend/features/job"
	"qurio/apps/backend/internal/middleware"
	"qurio/apps/backend/internal/text"
	"qurio/apps/backend/internal/config"
)

type PageDTO struct {
	SourceID string
	URL      string
	Status   string
	Depth    int
}

type PageManager interface {
	BulkCreatePages(ctx context.Context, pages []PageDTO) ([]string, error)
	UpdatePageStatus(ctx context.Context, sourceID, url, status, err string) error
	CountPendingPages(ctx context.Context, sourceID string) (int, error)
}

type TaskPublisher interface {
	Publish(topic string, body []byte) error
}

type ResultConsumer struct {
	store         VectorStore
	updater       SourceStatusUpdater
	jobRepo       job.Repository
	sourceFetcher SourceFetcher
	pageManager   PageManager
	publisher     TaskPublisher
}

func NewResultConsumer(s VectorStore, u SourceStatusUpdater, j job.Repository, sf SourceFetcher, pm PageManager, tp TaskPublisher) *ResultConsumer {
	return &ResultConsumer{
		store:         s,
		updater:       u,
		jobRepo:       j,
		sourceFetcher: sf,
		pageManager:   pm,
		publisher:     tp,
	}
}

func (h *ResultConsumer) HandleMessage(m *nsq.Message) error {
	if len(m.Body) == 0 {
		return nil
	}

	var payload struct {
		SourceID        string                 `json:"source_id"`
		Content         string                 `json:"content"`
		Title           string                 `json:"title"`
		Path            string                 `json:"path"`
		URL             string                 `json:"url"`
		Status          string                 `json:"status,omitempty"` // "success" or "failed"
		Error           string                 `json:"error,omitempty"`
		Links           []string               `json:"links,omitempty"`
		Depth           int                    `json:"depth"`
		CorrelationID   string                 `json:"correlation_id,omitempty"`
		OriginalPayload json.RawMessage        `json:"original_payload,omitempty"`
		Metadata        map[string]interface{} `json:"metadata,omitempty"`
	}

	err := json.Unmarshal(m.Body, &payload)

	correlationID := payload.CorrelationID
	if correlationID == "" {
		correlationID = uuid.New().String()
	}

	ctx := context.Background()
	ctx = middleware.WithCorrelationID(ctx, correlationID)

	if err != nil {
		slog.ErrorContext(ctx, "invalid message format", "error", err)
		return nil // Don't retry invalid messages
	}

	if payload.SourceID == "" || payload.URL == "" {
		slog.ErrorContext(ctx, "missing required fields, dropping", "source_id", payload.SourceID, "url", payload.URL)
		return nil
	}

	// Handle Failure
	if payload.Status == "failed" {
		slog.ErrorContext(ctx, "ingestion failed", "source_id", payload.SourceID, "url", payload.URL, "error", payload.Error)

		// Update Page Status
		if payload.URL != "" {
			_ = h.pageManager.UpdatePageStatus(ctx, payload.SourceID, payload.URL, "failed", payload.Error)
		}

		// Check if we should fail the source (maybe not? individual page failure shouldn't fail source?)
		// For now, let's keep the source "in_progress" but log the failure.
		// If it was the SEED page, maybe fail source?
		if payload.Depth == 0 {
			if err := h.updater.UpdateStatus(ctx, payload.SourceID, "failed"); err != nil {
				slog.WarnContext(ctx, "failed to update source status to failed", "error", err)
			}
		}

		// Save Failed Job
		if payload.OriginalPayload != nil {
			failedJob := &job.Job{
				SourceID: payload.SourceID,
				Handler:  "ingestion-worker", // Identify where it failed
				Payload:  payload.OriginalPayload,
				Error:    payload.Error,
			}
			if err := h.jobRepo.Save(ctx, failedJob); err != nil {
				slog.ErrorContext(ctx, "failed to save failed job", "error", err)
				// Don't return error here, we don't want to retry the result processing loop
			} else {
				slog.InfoContext(ctx, "saved failed job for retry", "job_id", failedJob.ID)
			}
		}

		return nil
	}

	slog.InfoContext(ctx, "received result", "source_id", payload.SourceID, "url", payload.URL, "content_len", len(payload.Content))

	// Fetch Source Config & Name
	maxDepth, exclusions, apiKey, sourceName, err := h.sourceFetcher.GetSourceConfig(ctx, payload.SourceID)
	if err != nil {
		slog.WarnContext(ctx, "failed to fetch source config", "error", err)
	}
	
	// 1. Delete Old Chunks (Idempotency)
	if payload.URL != "" {
		if err := h.store.DeleteChunksByURL(ctx, payload.SourceID, payload.URL); err != nil {
			slog.ErrorContext(ctx, "failed to delete old chunks", "error", err)
			return err 
		}
	}

	// 2. Chunk and Publish
	if payload.Content != "" {
		chunks := text.ChunkMarkdown(payload.Content, 512, 50)
		if len(chunks) > 0 {
			for i, c := range chunks {
				// Construct IngestEmbedPayload
				embedPayload := IngestEmbedPayload{
					SourceID:      payload.SourceID,
					SourceURL:     payload.URL,
					SourceName:    sourceName,
					Title:         payload.Title,
					Path:          payload.Path,
					
					Content:       c.Content,
					ChunkIndex:    i,
					ChunkType:     string(c.Type),
					Language:      c.Language,
					
					CorrelationID: correlationID,
				}

				if author, ok := payload.Metadata["author"].(string); ok {
					embedPayload.Author = author
				}
				if created, ok := payload.Metadata["created_at"].(string); ok {
					embedPayload.CreatedAt = created
				}
				if pages, ok := payload.Metadata["pages"].(float64); ok {
					embedPayload.PageCount = int(pages)
				}
				
				bytes, err := json.Marshal(embedPayload)
				if err != nil {
					slog.ErrorContext(ctx, "failed to marshal embed payload", "error", err)
					continue
				}

				if err := h.publisher.Publish(config.TopicIngestEmbed, bytes); err != nil {
					slog.ErrorContext(ctx, "failed to publish to ingest.embed", "error", err)
					return err // Durable: Fail if publish fails
				}
			}
			slog.InfoContext(ctx, "published embedding tasks", "count", len(chunks))
		}
	}

	// 3. Update Source Body Hash (Only for seed? Or aggregate? Maybe just last update)
	hash := sha256.Sum256([]byte(payload.Content))
	hashStr := fmt.Sprintf("%x", hash)
	_ = h.updater.UpdateBodyHash(ctx, payload.SourceID, hashStr)

	// 4. Distributed Crawl: Link Discovery
	if payload.URL != "" && len(payload.Links) > 0 {
		{
			u, _ := url.Parse(payload.URL)
			host := u.Host
			
			// Virtual Depth for llms.txt: Treat it as having +1 depth allowance
			effectiveMaxDepth := maxDepth
			isManifest := false
			if len(payload.URL) > 8 && payload.URL[len(payload.URL)-8:] == "llms.txt" {
				effectiveMaxDepth = maxDepth + 1
				isManifest = true
				slog.InfoContext(ctx, "processing manifest links with extended depth", "url", payload.URL)
			}

			newPages := DiscoverLinks(payload.SourceID, host, payload.Links, payload.Depth, effectiveMaxDepth, exclusions)
			
			if len(newPages) > 0 {
				newURLs, err := h.pageManager.BulkCreatePages(ctx, newPages)
				if err != nil {
					slog.ErrorContext(ctx, "failed to bulk create pages", "error", err)
				} else {
					slog.InfoContext(ctx, "discovered new pages", "count", len(newURLs))
					for _, newURL := range newURLs {
						// Ensure tasks generated from llms.txt at maxDepth don't exceed maxDepth+1 endlessly
						// Actually, DiscoverLinks sets new page depth as parent.Depth + 1.
						// If parent is llms.txt (depth=maxDepth), child will be maxDepth+1.
						// The child won't discover further links because its depth > maxDepth.
						// This is exactly what we want (1 level deeper than max).
						
						taskPayload, _ := json.Marshal(map[string]interface{}{
							"type":           "web",
							"url":            newURL,
							"id":             payload.SourceID,
							"depth":          payload.Depth + 1,
							"max_depth":      maxDepth,
							"exclusions":     exclusions,
							"gemini_api_key": apiKey,
							"correlation_id": correlationID,
						})
						if err := h.publisher.Publish(config.TopicIngestWeb, taskPayload); err != nil {
							slog.ErrorContext(ctx, "failed to publish task, marking page as failed", "error", err, "url", newURL)
							_ = h.pageManager.UpdatePageStatus(ctx, payload.SourceID, newURL, "failed", fmt.Sprintf("Failed to publish task: %v", err))
						}
					}
				}
			} else if isManifest {
				slog.InfoContext(ctx, "no new pages discovered from manifest (might be duplicates or excluded)", "url", payload.URL)
			}
		}
	}

	// 5. Update Page Status to Completed (Coordinator considers it done once chunks are queued)
	if payload.URL != "" {
		if err := h.pageManager.UpdatePageStatus(ctx, payload.SourceID, payload.URL, "completed", ""); err != nil {
			slog.WarnContext(ctx, "failed to update page status", "error", err)
		}
	}

	// 6. Check Source Completion

	pendingCount, err := h.pageManager.CountPendingPages(ctx, payload.SourceID)
	if err != nil {
		slog.WarnContext(ctx, "failed to count pending pages", "error", err)
	} else if pendingCount == 0 {
		slog.InfoContext(ctx, "source ingestion completed", "source_id", payload.SourceID)
		if err := h.updater.UpdateStatus(ctx, payload.SourceID, "completed"); err != nil {
			slog.WarnContext(ctx, "failed to update source status to completed", "error", err)
		}
	}
	
	return nil
}