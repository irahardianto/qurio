package settings_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"qurio/apps/backend/internal/settings"
)

// MockRepository is a mock implementation of settings.Repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Get(ctx context.Context) (*settings.Settings, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*settings.Settings), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, s *settings.Settings) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}

func TestHandler_GetSettings(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := settings.NewService(mockRepo)
		handler := settings.NewHandler(svc)

		expectedSettings := &settings.Settings{
			RerankProvider: "cohere",
			SearchAlpha:    0.5,
		}

		mockRepo.On("Get", mock.Anything).Return(expectedSettings, nil)

		req := httptest.NewRequest("GET", "/settings", nil)
		w := httptest.NewRecorder()

		handler.GetSettings(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)

		data := body["data"].(map[string]interface{})
		assert.Equal(t, "cohere", data["rerank_provider"])
		assert.Equal(t, 0.5, data["search_alpha"])

		mockRepo.AssertExpectations(t)
	})

	t.Run("InternalError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := settings.NewService(mockRepo)
		handler := settings.NewHandler(svc)

		mockRepo.On("Get", mock.Anything).Return(nil, errors.New("db error"))

		req := httptest.NewRequest("GET", "/settings", nil)
		w := httptest.NewRecorder()

		handler.GetSettings(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

func TestHandler_UpdateSettings(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := settings.NewService(mockRepo)
		handler := settings.NewHandler(svc)

		newSettings := &settings.Settings{
			RerankProvider: "jina",
			SearchAlpha:    0.7,
		}

		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(s *settings.Settings) bool {
			return s.RerankProvider == "jina" && s.SearchAlpha == 0.7
		})).Return(nil)

		body, _ := json.Marshal(newSettings)
		req := httptest.NewRequest("PUT", "/settings", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.UpdateSettings(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		mockRepo.AssertExpectations(t)
	})

	t.Run("ValidationError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := settings.NewService(mockRepo)
		handler := settings.NewHandler(svc)

		req := httptest.NewRequest("PUT", "/settings", bytes.NewBufferString("invalid json"))
		w := httptest.NewRecorder()

		handler.UpdateSettings(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})
}
