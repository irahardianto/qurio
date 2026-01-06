package job_test

import (
	"context"
	"database/sql"
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

// MockPublisher
type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) Publish(topic string, body []byte) error {
	args := m.Called(topic, body)
	return args.Error(0)
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

func TestHandler_Retry_NotFound(t *testing.T) {
	mockRepo := new(MockRepo)
	mockPub := new(MockPublisher)
	svc := job.NewService(mockRepo, mockPub, slog.Default())
	handler := job.NewHandler(svc)

	mockRepo.On("Get", mock.Anything, "99").Return(nil, sql.ErrNoRows)

	req := httptest.NewRequest("POST", "/jobs/99/retry", nil)
	req.SetPathValue("id", "99")
	w := httptest.NewRecorder()

	handler.Retry(w, req)
	assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
}

func TestHandler_Retry(t *testing.T) {
	mockRepo := new(MockRepo)
	mockPub := new(MockPublisher)
	svc := job.NewService(mockRepo, mockPub, slog.Default())
	handler := job.NewHandler(svc)

	jobID := "job-123"
	j := &job.Job{
		ID:      jobID,
		Payload: []byte(`{"url": "http://example.com"}`),
	}

	mockRepo.On("Get", mock.Anything, jobID).Return(j, nil)
	mockPub.On("Publish", "ingest.task", mock.Anything).Return(nil)
	mockRepo.On("Delete", mock.Anything, jobID).Return(nil)

	req := httptest.NewRequest("POST", "/jobs/"+jobID+"/retry", nil)
	req.SetPathValue("id", jobID)
	w := httptest.NewRecorder()

	handler.Retry(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	mockRepo.AssertExpectations(t)
	mockPub.AssertExpectations(t)
}