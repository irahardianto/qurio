package source

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"qurio/apps/backend/internal/worker"
	"qurio/apps/backend/internal/settings"
)

// Mock objects
type MockRepo struct {
	mock.Mock
}

// ... existing MockRepo methods ...

type MockSettingsService struct {
	mock.Mock
}

func (m *MockSettingsService) Get(ctx context.Context) (*settings.Settings, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*settings.Settings), args.Error(1)
}

func (m *MockRepo) Save(ctx context.Context, src *Source) error {
	args := m.Called(ctx, src)
	return args.Error(0)
}
func (m *MockRepo) ExistsByHash(ctx context.Context, hash string) (bool, error) {
	args := m.Called(ctx, hash)
	return args.Bool(0), args.Error(1)
}
func (m *MockRepo) Get(ctx context.Context, id string) (*Source, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*Source), args.Error(1)
}
func (m *MockRepo) List(ctx context.Context) ([]Source, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Source), args.Error(1)
}
func (m *MockRepo) UpdateStatus(ctx context.Context, id, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}
func (m *MockRepo) UpdateBodyHash(ctx context.Context, id, hash string) error {
	args := m.Called(ctx, id, hash)
	return args.Error(0)
}
func (m *MockRepo) SoftDelete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockPub struct {
	mock.Mock
}

func (m *MockPub) Publish(topic string, body []byte) error {
	args := m.Called(topic, body)
	return args.Error(0)
}

type MockChunkStore struct {
	mock.Mock
}

func (m *MockChunkStore) GetChunks(ctx context.Context, sourceID string) ([]worker.Chunk, error) {
	args := m.Called(ctx, sourceID)
	return args.Get(0).([]worker.Chunk), args.Error(1)
}

func TestCreate_FullPayload(t *testing.T) {
	repo := new(MockRepo)
	pub := new(MockPub)
	chunkStore := new(MockChunkStore)
	settingsMock := new(MockSettingsService)
	svc := NewService(repo, pub, chunkStore, settingsMock)
	handler := NewHandler(svc)

	body := map[string]interface{}{
		"url":        "http://example.com",
		"max_depth":  2,
		"exclusions": []string{"/blog", "/login"},
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/sources", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	repo.On("ExistsByHash", mock.Anything, mock.Anything).Return(false, nil)
	repo.On("Save", mock.Anything, mock.MatchedBy(func(src *Source) bool {
		return src.URL == "http://example.com" &&
			src.MaxDepth == 2 &&
			len(src.Exclusions) == 2 &&
			src.Exclusions[0] == "/blog"
	})).Return(nil)
	
	settingsMock.On("Get", mock.Anything).Return(&settings.Settings{}, nil)

	// Verify "max_depth" key in published payload
	pub.On("Publish", "ingest.task", mock.MatchedBy(func(payload []byte) bool {
		var p map[string]interface{}
		json.Unmarshal(payload, &p)
		// Check that "max_depth" key exists and matches
		return p["max_depth"] == float64(2) &&
			p["url"] == "http://example.com"
	})).Return(nil)

	handler.Create(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotNil(t, resp["data"])

	repo.AssertExpectations(t)
	pub.AssertExpectations(t)
}
