package gemini_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/option"

	"qurio/apps/backend/internal/adapter/gemini"
	"qurio/apps/backend/internal/settings"
)

// MockSettingsRepo
type MockSettingsRepo struct {
	mock.Mock
}

func (m *MockSettingsRepo) Get(ctx context.Context) (*settings.Settings, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*settings.Settings), args.Error(1)
}

func (m *MockSettingsRepo) Update(ctx context.Context, s *settings.Settings) error {
	return m.Called(ctx, s).Error(0)
}

func TestDynamicEmbedder_Embed(t *testing.T) {
	// Mock Settings
	mockRepo := new(MockSettingsRepo)
	settingsSvc := settings.NewService(mockRepo)

	// Mock Gemini Server
	// The client will likely append /v1beta/... or similar depending on internal logic.
	// We catch all for now to see what hits.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return a dummy embedding response
		json.NewEncoder(w).Encode(map[string]interface{}{
			"embedding": map[string]interface{}{
				"values": []float32{0.1, 0.2, 0.3},
			},
		})
	}))
	defer ts.Close()

	// Initialize Embedder with options
	embedder := gemini.NewDynamicEmbedder(
		settingsSvc,
		option.WithEndpoint(ts.URL),
	)

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Get", ctx).Return(&settings.Settings{GeminiAPIKey: "test-key"}, nil).Once()

		vec, err := embedder.Embed(ctx, "hello world")
		assert.NoError(t, err)
		if assert.Len(t, vec, 3) {
			assert.Equal(t, float32(0.1), vec[0])
		}
		mockRepo.AssertExpectations(t)
	})

	t.Run("Missing API Key", func(t *testing.T) {
		mockRepo.On("Get", ctx).Return(&settings.Settings{GeminiAPIKey: ""}, nil).Once()

		vec, err := embedder.Embed(ctx, "hello")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gemini api key not configured")
		assert.Nil(t, vec)
		mockRepo.AssertExpectations(t)
	})
}
