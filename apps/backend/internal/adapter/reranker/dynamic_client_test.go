package reranker

import (
	"context"
	"testing"

	"qurio/apps/backend/internal/settings"

	"github.com/stretchr/testify/assert"
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

func TestDynamicClient_Rerank_SettingsError(t *testing.T) {
	repo := &MockSettingsRepo{
		Settings: nil,
		Err:      assert.AnError,
	}
	svc := settings.NewService(repo)
	client := NewDynamicClient(svc)

	_, err := client.Rerank(context.Background(), "query", []string{"doc1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get settings")
}

func TestDynamicClient_GetClient_Caching(t *testing.T) {
	dc := NewDynamicClient(nil) // settingsSvc not needed for getClient

	// First call creates client
	c1 := dc.getClient("jina", "key-1")
	assert.NotNil(t, c1)

	// Same params returns cached client
	c2 := dc.getClient("jina", "key-1")
	assert.Equal(t, c1, c2, "should return same cached client")

	// Different key recreates client
	c3 := dc.getClient("jina", "key-2")
	assert.NotNil(t, c3)
	assert.NotEqual(t, c1, c3, "should create new client for different key")

	// Different provider recreates client
	c4 := dc.getClient("cohere", "key-2")
	assert.NotNil(t, c4)
	assert.NotEqual(t, c3, c4, "should create new client for different provider")
}
