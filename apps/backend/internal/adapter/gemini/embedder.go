package gemini

import (
	"context"
	"log/slog"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type Embedder struct {
	client *genai.Client
	model  string
}

func NewEmbedder(ctx context.Context, apiKey string) (*Embedder, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	return &Embedder{client: client, model: "gemini-embedding-001"}, nil
}

func (e *Embedder) Embed(ctx context.Context, text string) ([]float32, error) {
	slog.DebugContext(ctx, "embedding content", "model", e.model, "length", len(text))
	em := e.client.EmbeddingModel(e.model)
	res, err := em.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		slog.ErrorContext(ctx, "embedding failed", "error", err)
		return nil, err
	}
	if res.Embedding != nil {
		return res.Embedding.Values, nil
	}
	return nil, nil
}
