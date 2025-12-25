package worker

import (
	"context"
)

type Chunk struct {
	Content    string
	Vector     []float32
	SourceURL  string
	SourceID   string
	ChunkIndex int
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
