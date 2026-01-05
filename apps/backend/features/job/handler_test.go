package job_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"qurio/apps/backend/features/job"
)

// MockRepo implements job.Repository
type MockRepo struct {
	mock.Mock
}

func (m *MockRepo) Save(ctx context.Context, j *job.Job) error {
	args := m.Called(ctx, j)
	return args.Error(0)
}
func (m *MockRepo) List(ctx context.Context) ([]job.Job, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]job.Job), args.Error(1)
}
func (m *MockRepo) Get(ctx context.Context, id string) (*job.Job, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*job.Job), args.Error(1)
}
func (m *MockRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockRepo) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func TestHandler_List(t *testing.T) {
	mockRepo := new(MockRepo)
	svc := job.NewService(mockRepo, nil, slog.Default()) // nil nsq
	handler := job.NewHandler(svc)

	mockRepo.On("List", mock.Anything).Return([]job.Job{}, nil)

	req := httptest.NewRequest("GET", "/jobs", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestHandler_Retry(t *testing.T) {
	// Skip Retry test due to NSQ dependency
}
