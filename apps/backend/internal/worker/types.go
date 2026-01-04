package worker

import (
	"context"
)

type Chunk struct {
	Content    string    `json:"content"`
	Vector     []float32 `json:"vector"`
	SourceURL  string    `json:"source_url"`
	SourceID   string    `json:"source_id"`
	SourceName string    `json:"source_name"`
	ChunkIndex int       `json:"chunk_index"`
	Type       string    `json:"type"`
	Language   string    `json:"language"`
	Title      string    `json:"title"`
	Author     string    `json:"author"`
	CreatedAt  string    `json:"created_at"`
	PageCount  int       `json:"page_count"`
}

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type VectorStore interface {
	StoreChunk(ctx context.Context, chunk Chunk) error
	DeleteChunksByURL(ctx context.Context, sourceID, url string) error
}

type SourceStatusUpdater interface {
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateBodyHash(ctx context.Context, id, hash string) error
}

type SourceFetcher interface {
	GetSourceDetails(ctx context.Context, id string) (string, string, error)
	GetSourceConfig(ctx context.Context, id string) (int, []string, string, string, error)
}
