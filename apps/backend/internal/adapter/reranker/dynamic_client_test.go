package reranker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"qurio/apps/backend/internal/settings"
)

type MockSettingsRepo struct {
	Settings *settings.Settings
	Err      error
}

func (m *MockSettingsRepo) Get(ctx context.Context) (*settings.Settings, error) {
	return m.Settings, m.Err
}

func (m *MockSettingsRepo) Update(ctx context.Context, s *settings.Settings) error {
	return nil
}

func TestDynamicClient_Rerank_None(t *testing.T) {
	repo := &MockSettingsRepo{
		Settings: &settings.Settings{
			RerankProvider: "none",
		},
	}
	svc := settings.NewService(repo)
	client := NewDynamicClient(svc)

	docs := []string{"doc1", "doc2"}
	indices, err := client.Rerank(context.Background(), "query", docs)

	assert.NoError(t, err)
	assert.Equal(t, []int{0, 1}, indices)
}

func TestDynamicClient_Rerank_EmptyProvider(t *testing.T) {
	repo := &MockSettingsRepo{
		Settings: &settings.Settings{
			RerankProvider: "",
		},
	}
	svc := settings.NewService(repo)
	client := NewDynamicClient(svc)

	docs := []string{"doc1", "doc2"}
	indices, err := client.Rerank(context.Background(), "query", docs)

	assert.NoError(t, err)
	assert.Equal(t, []int{0, 1}, indices)
}
