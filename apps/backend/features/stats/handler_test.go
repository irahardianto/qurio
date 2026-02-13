package stats

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockSourceRepo struct{ mock.Mock }

func (m *MockSourceRepo) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

type MockJobRepo struct{ mock.Mock }

func (m *MockJobRepo) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

type MockVectorStore struct{ mock.Mock }

func (m *MockVectorStore) CountChunks(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func TestHandler_GetStats_Table(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockSourceRepo, *MockJobRepo, *MockVectorStore)
		wantStatus int
		wantError  bool
		checkBody  func(*testing.T, map[string]interface{})
	}{
		{
			name: "Success",
			setupMocks: func(s *MockSourceRepo, j *MockJobRepo, v *MockVectorStore) {
				s.On("Count", mock.Anything).Return(10, nil)
				j.On("Count", mock.Anything).Return(5, nil)
				v.On("CountChunks", mock.Anything).Return(100, nil)
			},
			wantStatus: http.StatusOK,
			wantError:  false,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				data := body["data"].(map[string]interface{})
				assert.EqualValues(t, 10, data["sources"])
				assert.EqualValues(t, 5, data["failed_jobs"])
				assert.EqualValues(t, 100, data["documents"])
			},
		},
		{
			name: "SourceRepo Error",
			setupMocks: func(s *MockSourceRepo, j *MockJobRepo, v *MockVectorStore) {
				s.On("Count", mock.Anything).Return(0, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  true,
		},
		{
			name: "JobRepo Error",
			setupMocks: func(s *MockSourceRepo, j *MockJobRepo, v *MockVectorStore) {
				s.On("Count", mock.Anything).Return(10, nil)
				j.On("Count", mock.Anything).Return(0, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  true,
		},
		{
			name: "VectorStore Error",
			setupMocks: func(s *MockSourceRepo, j *MockJobRepo, v *MockVectorStore) {
				s.On("Count", mock.Anything).Return(10, nil)
				j.On("Count", mock.Anything).Return(5, nil)
				v.On("CountChunks", mock.Anything).Return(0, errors.New("weaviate error"))
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mSource := new(MockSourceRepo)
			mJob := new(MockJobRepo)
			mVector := new(MockVectorStore)

			tt.setupMocks(mSource, mJob, mVector)

			h := NewHandler(mSource, mJob, mVector)
			req := httptest.NewRequest("GET", "/stats", nil)
			w := httptest.NewRecorder()

			h.GetStats(w, req)

			resp := w.Result()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			var body map[string]interface{}
			err := json.NewDecoder(resp.Body).Decode(&body)
			assert.NoError(t, err)

			if tt.wantError {
				assert.Contains(t, body, "error")
				errMap := body["error"].(map[string]interface{})
				assert.Equal(t, "INTERNAL_ERROR", errMap["code"])
			} else {
				tt.checkBody(t, body)
			}
		})
	}
}
