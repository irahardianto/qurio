package source

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"

	"qurio/apps/backend/internal/worker"
	"qurio/apps/backend/internal/settings"
	"qurio/apps/backend/internal/middleware"
)

type Source struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	URL         string   `json:"url"`
	ContentHash string   `json:"-"`
	BodyHash    string   `json:"-"`
	Status      string   `json:"status"`
	MaxDepth    int      `json:"max_depth"`
	Exclusions  []string `json:"exclusions"`
}

type SourcePage struct {
	ID        string `json:"id"`
	SourceID  string `json:"source_id"`
	URL       string `json:"url"`
	Status    string `json:"status"` // pending, processing, completed, failed
	Depth     int    `json:"depth"`
	Error     string `json:"error,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Repository interface {
	// Pages
	BulkCreatePages(ctx context.Context, pages []SourcePage) ([]string, error)
	UpdatePageStatus(ctx context.Context, sourceID, url, status, err string) error
	GetPages(ctx context.Context, sourceID string) ([]SourcePage, error)
	DeletePages(ctx context.Context, sourceID string) error
	CountPendingPages(ctx context.Context, sourceID string) (int, error)
	
	// Sources

	Save(ctx context.Context, src *Source) error
	ExistsByHash(ctx context.Context, hash string) (bool, error)
	Get(ctx context.Context, id string) (*Source, error)
	List(ctx context.Context) ([]Source, error)
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateBodyHash(ctx context.Context, id, hash string) error
	SoftDelete(ctx context.Context, id string) error
	Count(ctx context.Context) (int, error)
}

type ChunkStore interface {
	GetChunks(ctx context.Context, sourceID string) ([]worker.Chunk, error)
	DeleteChunksBySourceID(ctx context.Context, sourceID string) error
}

type EventPublisher interface {
	Publish(topic string, body []byte) error
}

type SettingsService interface {
	Get(ctx context.Context) (*settings.Settings, error)
}

type Service struct {
	repo       Repository
	pub        EventPublisher
	chunkStore ChunkStore
	settings   SettingsService
}

func NewService(repo Repository, pub EventPublisher, chunkStore ChunkStore, settings SettingsService) *Service {
	return &Service{repo: repo, pub: pub, chunkStore: chunkStore, settings: settings}
}

func (s *Service) Create(ctx context.Context, src *Source) error {
	// 0. Compute Hash
	hash := sha256.Sum256([]byte(src.URL))
	src.ContentHash = fmt.Sprintf("%x", hash)

	// Default to web if empty
	if src.Type == "" {
		src.Type = "web"
	}

	// 1. Check Duplicate
	exists, err := s.repo.ExistsByHash(ctx, src.ContentHash)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("Duplicate detected")
	}

	// 2. Set Status to in_progress (queued) and Save
	src.Status = "in_progress"
	if err := s.repo.Save(ctx, src); err != nil {
		return err
	}

	// 2.1 Create Seed Page (Crawl Frontier)
	if src.Type == "web" {
		_, err = s.repo.BulkCreatePages(ctx, []SourcePage{{
			SourceID: src.ID,
			URL:      src.URL,
			Status:   "pending",
			Depth:    0,
		}})
		if err != nil {
			// Log error but proceed? No, fail.
			return fmt.Errorf("failed to create seed page: %w", err)
		}
	}

	// 3. Get Settings
	set, err := s.settings.Get(ctx)
	apiKey := ""
	if err == nil && set != nil {
		apiKey = set.GeminiAPIKey
	}

	// 4. Publish to NSQ
	payload, _ := json.Marshal(map[string]interface{}{
		"type":           src.Type,
		"url":            src.URL,
		"id":             src.ID,
		"depth":          0, // Seed depth
		"max_depth":      src.MaxDepth,
		"exclusions":     src.Exclusions,
		"gemini_api_key": apiKey,
		"correlation_id": middleware.GetCorrelationID(ctx),
	})
	if err := s.pub.Publish("ingest.task", payload); err != nil {
		slog.Error("failed to publish ingest.task event", "error", err)
	} else {
		slog.Info("published ingest.task event", "url", src.URL, "id", src.ID)
	}
	
	return nil
}

func (s *Service) Upload(ctx context.Context, path string, hash string) (*Source, error) {
	// Check Duplicate
	exists, err := s.repo.ExistsByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("Duplicate detected")
	}

	src := &Source{
		Type:        "file",
		URL:         path, // Use URL field to store file path
		ContentHash: hash,
		Status:      "in_progress",
	}

	if err := s.repo.Save(ctx, src); err != nil {
		return nil, err
	}

	// Publish to NSQ
	payload, _ := json.Marshal(map[string]interface{}{
		"type":           "file",
		"path":           path,
		"id":             src.ID,
		"correlation_id": middleware.GetCorrelationID(ctx),
	})
	if err := s.pub.Publish("ingest.task", payload); err != nil {
		slog.Error("failed to publish ingest.task event (upload)", "error", err)
	} else {
		slog.Info("published ingest.task event (upload)", "path", path, "id", src.ID)
	}

	return src, nil
}

type SourceDetail struct {
	Source
	Chunks      []worker.Chunk `json:"chunks"`
	TotalChunks int            `json:"total_chunks"`
}

func (s *Service) Get(ctx context.Context, id string) (*SourceDetail, error) {
	src, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	chunks, err := s.chunkStore.GetChunks(ctx, id)
	if err != nil {
		slog.Warn("failed to fetch chunks", "error", err, "source_id", id)
		chunks = []worker.Chunk{}
	}

	return &SourceDetail{
		Source:      *src,
		Chunks:      chunks,
		TotalChunks: len(chunks),
	}, nil
}

func (s *Service) List(ctx context.Context) ([]Source, error) {
	return s.repo.List(ctx)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	// 1. Clean Vector Store
	if err := s.chunkStore.DeleteChunksBySourceID(ctx, id); err != nil {
		return err
	}
	// 2. Soft Delete DB
	return s.repo.SoftDelete(ctx, id)
}

func (s *Service) ReSync(ctx context.Context, id string) error {
	src, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	// Update Status to in_progress
	if err := s.repo.UpdateStatus(ctx, id, "in_progress"); err != nil {
		return err
	}

	// Clean up pages for fresh start
	if src.Type == "web" {
		if err := s.repo.DeletePages(ctx, id); err != nil {
			return fmt.Errorf("failed to clean up pages: %w", err)
		}
		// Re-create Seed Page
		_, err = s.repo.BulkCreatePages(ctx, []SourcePage{{
			SourceID: src.ID,
			URL:      src.URL,
			Status:   "pending",
			Depth:    0,
		}})
		if err != nil {
			return fmt.Errorf("failed to recreate seed page: %w", err)
		}
	}

	set, err := s.settings.Get(ctx)
	apiKey := ""
	if err == nil && set != nil {
		apiKey = set.GeminiAPIKey
	}

	payloadMap := map[string]interface{}{
		"type":           src.Type,
		"id":             src.ID,
		"resync":         true,
		"correlation_id": middleware.GetCorrelationID(ctx),
	}

	if src.Type == "file" {
		payloadMap["path"] = src.URL
	} else {
		payloadMap["url"] = src.URL
		payloadMap["depth"] = 0 // Reset depth
		payloadMap["max_depth"] = src.MaxDepth
		payloadMap["exclusions"] = src.Exclusions
		payloadMap["gemini_api_key"] = apiKey
	}

	payload, _ := json.Marshal(payloadMap)
	if err := s.pub.Publish("ingest.task", payload); err != nil {
		slog.Error("failed to publish resync event", "error", err)
		return err
	}
	return nil
}

func (s *Service) GetPages(ctx context.Context, id string) ([]SourcePage, error) {
	return s.repo.GetPages(ctx, id)
}
