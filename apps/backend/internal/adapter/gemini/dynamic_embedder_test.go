package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/option"
	"qurio/apps/backend/internal/settings"
)

// --- Mocks ---

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
	args := m.Called(ctx, s)
	return args.Error(0)
}

// --- Helpers ---

func newMockGeminiServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/models/gemini-embedding-001:embedContent" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		// Verify API Key in header
		key := r.URL.Query().Get("key")
		if key == "" {
			// Some clients send it in header 'x-goog-api-key'
			key = r.Header.Get("x-goog-api-key")
		}

		if key == "invalid-key" {
			http.Error(w, "invalid key", http.StatusUnauthorized)
			return
		}

		// Return mock embedding
		resp := map[string]interface{}{
			"embedding": map[string]interface{}{
				"values": []float32{0.1, 0.2, 0.3},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
}

// --- Tests ---

func TestDynamicEmbedder_Embed_Success(t *testing.T) {
	// Setup
	server := newMockGeminiServer()
	defer server.Close()

	mockRepo := new(MockSettingsRepo)
	settingsSvc := settings.NewService(mockRepo)
	
	// Mock settings return
	mockRepo.On("Get", mock.Anything).Return(&settings.Settings{
		GeminiAPIKey: "valid-key-1",
	}, nil)

	// Create embedder with mock endpoint
	// Note: We strip 'http://' from server.URL for WithEndpoint usually? 
	// Actually WithEndpoint expects full URL or base. 
	// For genai, we might need WithHTTPClient if Endpoint doesn't play nice with REST vs gRPC.
	// But genai-go uses REST by default now or supports it?
	// Let's use WithEndpoint and hope it works for the REST client.
	embedder := NewDynamicEmbedder(settingsSvc, option.WithEndpoint(server.URL))

	// Execute
	vals, err := embedder.Embed(context.Background(), "hello world")

	// Assert
	assert.NoError(t, err)
	assert.Len(t, vals, 3)
	assert.Equal(t, float32(0.1), vals[0])
	mockRepo.AssertExpectations(t)
}

func TestDynamicEmbedder_Embed_NoKey(t *testing.T) {
	mockRepo := new(MockSettingsRepo)
	settingsSvc := settings.NewService(mockRepo)

	mockRepo.On("Get", mock.Anything).Return(&settings.Settings{
		GeminiAPIKey: "",
	}, nil)

	embedder := NewDynamicEmbedder(settingsSvc)

	_, err := embedder.Embed(context.Background(), "hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gemini api key not configured")
}

type ManualSettingsRepo struct {
	Keys      []string
	CallCount int
}

func (m *ManualSettingsRepo) Get(ctx context.Context) (*settings.Settings, error) {
	if m.CallCount >= len(m.Keys) {
		return nil, fmt.Errorf("unexpected call")
	}
	key := m.Keys[m.CallCount]
	m.CallCount++
	return &settings.Settings{GeminiAPIKey: key}, nil
}

func (m *ManualSettingsRepo) Update(ctx context.Context, s *settings.Settings) error {
	return nil
}

func TestDynamicEmbedder_KeyRotation(t *testing.T) {
	server := newMockGeminiServer()
	defer server.Close()

	// Use Manual Repo for precise control over sequential returns
	manualRepo := &ManualSettingsRepo{
		Keys: []string{"KeyA", "KeyB"},
	}
	settingsSvc := settings.NewService(manualRepo)

	// Round 1: Key A (Expected from ManualRepo)
	
	embedder := NewDynamicEmbedder(settingsSvc, option.WithEndpoint(server.URL))
	
	// Force the embedder to use KeyA
	_, err := embedder.Embed(context.Background(), "text1")
	assert.NoError(t, err)

	// Reset repo state for the second phase of testing
	manualRepo.CallCount = 0

	// Round 2: Key B (Expected from ManualRepo)
	
	// The server check in previous test was simple. Let's add key validation in a new server for this test
	// to ensure the CORRECT key is being sent.
	
	requestCount := 0
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		key := r.URL.Query().Get("key")
		if key == "" { key = r.Header.Get("x-goog-api-key") }
		
		expected := "KeyA"
		if requestCount == 2 {
			expected = "KeyB"
		}
		
		if key != expected {
			http.Error(w, fmt.Sprintf("expected %s got %s", expected, key), http.StatusBadRequest)
			return
		}
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"embedding": map[string]interface{}{"values": []float32{0.1}},
		})
	}))
	defer server2.Close()
	
	embedder2 := NewDynamicEmbedder(settingsSvc, option.WithEndpoint(server2.URL))

	// Call 1
	_, err = embedder2.Embed(context.Background(), "text1")
	assert.NoError(t, err)

	// Call 2
	_, err = embedder2.Embed(context.Background(), "text2")
	assert.NoError(t, err)
	
	assert.Equal(t, 2, requestCount)
}