package gemini

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"qurio/apps/backend/internal/settings"
)

// MockRepo implements settings.Repository
type MockRepo struct {
	Settings *settings.Settings
	Err      error
}

func (m *MockRepo) Get(ctx context.Context) (*settings.Settings, error) {
	return m.Settings, m.Err
}

func (m *MockRepo) Update(ctx context.Context, s *settings.Settings) error {
	return nil
}

func TestDynamicEmbedder_Embed_NoKey(t *testing.T) {
	repo := &MockRepo{
		Settings: &settings.Settings{GeminiAPIKey: ""},
	}
	svc := settings.NewService(repo)
	embedder := NewDynamicEmbedder(svc)

	_, err := embedder.Embed(context.Background(), "hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gemini api key not configured")
}

func TestDynamicEmbedder_Embed_SettingsError(t *testing.T) {
	repo := &MockRepo{
		Err: errors.New("db fail"),
	}
	svc := settings.NewService(repo)
	embedder := NewDynamicEmbedder(svc)

	_, err := embedder.Embed(context.Background(), "hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get settings")
}

func TestDynamicEmbedder_ClientSwitching(t *testing.T) {
	repo := &MockRepo{
		Settings: &settings.Settings{GeminiAPIKey: "key1"},
	}
	svc := settings.NewService(repo)
	embedder := NewDynamicEmbedder(svc)

	// We can't easily test Embed() success without mocking Google API completely,
	// but we can test the client switching logic by inspecting private fields.
	
	ctx := context.Background()
	
	// First call - initializes client
	client1, err := embedder.getClient(ctx, "key1")
	assert.NoError(t, err)
	assert.NotNil(t, client1)
	assert.Equal(t, "key1", embedder.currentKey)

	// Second call - same key - should be same client
	client2, err := embedder.getClient(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, client1, client2)

	// Third call - different key - should be new client
	client3, err := embedder.getClient(ctx, "key2")
	assert.NoError(t, err)
	assert.NotEqual(t, client1, client3)
	assert.Equal(t, "key2", embedder.currentKey)
}
