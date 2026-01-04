package worker

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/nsqio/go-nsq"
	"qurio/apps/backend/features/job"
	"qurio/apps/backend/internal/middleware"
	"qurio/apps/backend/internal/text"
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
	embedder      Embedder
	store         VectorStore
	updater       SourceStatusUpdater
	jobRepo       job.Repository
	sourceFetcher SourceFetcher
	pageManager   PageManager
	publisher     TaskPublisher
}

func NewResultConsumer(e Embedder, s VectorStore, u SourceStatusUpdater, j job.Repository, sf SourceFetcher, pm PageManager, tp TaskPublisher) *ResultConsumer {
	return &ResultConsumer{
		embedder:      e,
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
	if err := json.Unmarshal(m.Body, &payload); err != nil {
		slog.Error("invalid message format", "error", err)
		return nil // Don't retry invalid messages
	}

	correlationID := payload.CorrelationID
	if correlationID == "" {
		correlationID = uuid.New().String()
	}

	ctx := context.Background()
	ctx = middleware.WithCorrelationID(ctx, correlationID)

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

	// 0. Update Page Status to Processing (or skip, just update to completed at end)
	
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

	// 2. Chunk, Embed, Store
	if payload.Content != "" {
		chunks := text.ChunkMarkdown(payload.Content, 512, 50)
		if len(chunks) > 0 {
			for i, c := range chunks {
				err := func() error {
					embedCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
					defer cancel()

					// Contextual Embedding (FR-06)
					// Embedding Format:
					// Title: <Page Title>
					// URL: <Page URL>
					// Type: <Content Type>
					// Author: <Author> (Optional)
					// Created: <Created At> (Optional)
					// ---
					// <Raw Chunk Content>
					contextualString := fmt.Sprintf("Title: %s\nSource: %s\nPath: %s\nURL: %s\nType: %s", 
						payload.Title, sourceName, payload.Path, payload.URL, string(c.Type))

					if author, ok := payload.Metadata["author"].(string); ok && author != "" {
						contextualString += fmt.Sprintf("\nAuthor: %s", author)
					}
					if created, ok := payload.Metadata["created_at"].(string); ok && created != "" {
						contextualString += fmt.Sprintf("\nCreated: %s", created)
					}

					contextualString += fmt.Sprintf("\n---\n%s", c.Content)

					vector, err := h.embedder.Embed(embedCtx, contextualString)
					if err != nil {
						return err
					}

					chunk := Chunk{
						Content:    c.Content, // Store original content (FR-08)
						Vector:     vector,
						SourceID:   payload.SourceID,
						SourceURL:  payload.URL,
						ChunkIndex: i,
						Type:       string(c.Type),
						Language:   c.Language,
						Title:      payload.Title,
						SourceName: sourceName,
					}

					if author, ok := payload.Metadata["author"].(string); ok {
						chunk.Author = author
					}
					if created, ok := payload.Metadata["created_at"].(string); ok {
						chunk.CreatedAt = created
					}
					if pages, ok := payload.Metadata["pages"].(float64); ok {
						chunk.PageCount = int(pages)
					}

					return h.store.StoreChunk(embedCtx, chunk)
				}()
				if err != nil {
					slog.ErrorContext(ctx, "store chunk failed", "error", err)
					return err
				}
			}
			slog.InfoContext(ctx, "stored chunks", "count", len(chunks))
		}
	}

	// 3. Update Source Body Hash (Only for seed? Or aggregate? Maybe just last update)
	hash := sha256.Sum256([]byte(payload.Content))
	hashStr := fmt.Sprintf("%x", hash)
	_ = h.updater.UpdateBodyHash(ctx, payload.SourceID, hashStr)

	// 4. Distributed Crawl: Link Discovery
	if payload.URL != "" && len(payload.Links) > 0 {
		if payload.Depth < maxDepth {
			var newPages []PageDTO
			seen := make(map[string]bool)
			
			u, _ := url.Parse(payload.URL)
			host := u.Host
			
			for _, link := range payload.Links {
				// 1. External Check
				linkU, err := url.Parse(link)
				if err != nil || linkU.Host != host {
					continue
				}

				// Normalize URL: Strip Fragment
				linkU.Fragment = ""
				normalizedLink := linkU.String()
				
				// 2. Exclusion Check
			excluded := false
			for _, ex := range exclusions {
				if matched, _ := regexp.MatchString(ex, normalizedLink); matched {
					excluded = true
					break
				}
			}
			if excluded {
				continue
			}
			
			if seen[normalizedLink] { continue }
			seen[normalizedLink] = true
			
				newPages = append(newPages, PageDTO{
					SourceID: payload.SourceID,
					URL:      normalizedLink,
					Status:   "pending",
					Depth:    payload.Depth + 1,
				})
			}
			
			if len(newPages) > 0 {
				newURLs, err := h.pageManager.BulkCreatePages(ctx, newPages)
				if err != nil {
					slog.ErrorContext(ctx, "failed to bulk create pages", "error", err)
				} else {
					slog.InfoContext(ctx, "discovered new pages", "count", len(newURLs))
					for _, newURL := range newURLs {
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
							if err := h.publisher.Publish("ingest.task", taskPayload); err != nil {
							slog.ErrorContext(ctx, "failed to publish task, marking page as failed", "error", err, "url", newURL)
							_ = h.pageManager.UpdatePageStatus(ctx, payload.SourceID, newURL, "failed", fmt.Sprintf("Failed to publish task: %v", err))
						}
					}
				}
			}
		}
	}

	// 5. Update Page Status to Completed
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
