package source

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"

	"qurio/apps/backend/internal/worker"
	"qurio/apps/backend/internal/settings"
)

type Source struct {
	ID          string   `json:"id"`
	URL         string   `json:"url"`
	ContentHash string   `json:"-"`
	BodyHash    string   `json:"-"`
	Status      string   `json:"status"`
	MaxDepth    int      `json:"max_depth"`
	Exclusions  []string `json:"exclusions"`
}

type Repository interface {
	Save(ctx context.Context, src *Source) error
	ExistsByHash(ctx context.Context, hash string) (bool, error)
	Get(ctx context.Context, id string) (*Source, error)
	List(ctx context.Context) ([]Source, error)
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateBodyHash(ctx context.Context, id, hash string) error
	SoftDelete(ctx context.Context, id string) error
}

type ChunkStore interface {
	GetChunks(ctx context.Context, sourceID string) ([]worker.Chunk, error)
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

	// 3. Get Settings
	set, err := s.settings.Get(ctx)
	apiKey := ""
	if err == nil && set != nil {
		apiKey = set.GeminiAPIKey
	}

	// 4. Publish to NSQ
	payload, _ := json.Marshal(map[string]interface{}{
		"type":           "web",
		"url":            src.URL,
		"id":             src.ID,
		"max_depth":      src.MaxDepth,
		"exclusions":     src.Exclusions,
		"gemini_api_key": apiKey,
	})
	if err := s.pub.Publish("ingest.task", payload); err != nil {
		slog.Error("failed to publish ingest.task event", "error", err)
	} else {
		slog.Info("published ingest.task event", "url", src.URL, "id", src.ID)
	}
	
	return nil
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

	set, err := s.settings.Get(ctx)
	apiKey := ""
	if err == nil && set != nil {
		apiKey = set.GeminiAPIKey
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"type":           "web",
		"url":            src.URL,
		"id":             src.ID,
		"resync":         true,
		"max_depth":      src.MaxDepth,
		"exclusions":     src.Exclusions,
		"gemini_api_key": apiKey,
	})
	if err := s.pub.Publish("ingest.task", payload); err != nil {
		slog.Error("failed to publish resync event", "error", err)
		return err
	}
	return nil
}
