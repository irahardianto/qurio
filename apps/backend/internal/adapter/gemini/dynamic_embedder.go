package gemini

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"qurio/apps/backend/internal/settings"
)

type DynamicEmbedder struct {
	settingsSvc *settings.Service
	client      *genai.Client
	currentKey  string
	mu          sync.RWMutex
	clientOpts  []option.ClientOption
}

func NewDynamicEmbedder(svc *settings.Service, opts ...option.ClientOption) *DynamicEmbedder {
	return &DynamicEmbedder{
		settingsSvc: svc,
		clientOpts:  opts,
	}
}

func (e *DynamicEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	s, err := e.settingsSvc.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	if s.GeminiAPIKey == "" {
		return nil, fmt.Errorf("gemini api key not configured")
	}

	client, err := e.getClient(ctx, s.GeminiAPIKey)
	if err != nil {
		return nil, err
	}

	model := client.EmbeddingModel("gemini-embedding-001")
	res, err := model.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, err
	}

	if len(res.Embedding.Values) == 0 {
		return nil, fmt.Errorf("empty embedding received")
	}

	return res.Embedding.Values, nil
}

func (e *DynamicEmbedder) getClient(ctx context.Context, key string) (*genai.Client, error) {
	e.mu.RLock()
	if e.client != nil && e.currentKey == key {
		defer e.mu.RUnlock()
		return e.client, nil
	}
	e.mu.RUnlock()

	e.mu.Lock()
	defer e.mu.Unlock()

	// Double check
	if e.client != nil && e.currentKey == key {
		return e.client, nil
	}

	if e.client != nil {
		if err := e.client.Close(); err != nil {
			slog.Warn("failed to close previous genai client", "error", err)
		}
	}

	opts := append(e.clientOpts, option.WithAPIKey(key))
	client, err := genai.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	e.client = client
	e.currentKey = key
	return client, nil
}
