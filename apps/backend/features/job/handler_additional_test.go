package job_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"qurio/apps/backend/features/job"
	"qurio/apps/backend/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandler_List_ServiceError(t *testing.T) {
	mockRepo := new(MockRepo)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	svc := job.NewService(mockRepo, nil, logger)
	handler := job.NewHandler(svc)

	mockRepo.On("List", mock.Anything).Return(nil, errors.New("database error"))

	req := httptest.NewRequest("GET", "/jobs", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	mockRepo.AssertExpectations(t)
}

func TestHandler_List_EmptyList(t *testing.T) {
	mockRepo := new(MockRepo)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	svc := job.NewService(mockRepo, nil, logger)
	handler := job.NewHandler(svc)

	// Return nil slice
	mockRepo.On("List", mock.Anything).Return(nil, nil)

	req := httptest.NewRequest("GET", "/jobs", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	// Should allow parsing response even if empty
	// The implementation handles nil jobs by returning empty slice or nil (which marshals to null or [])
	// Implementation says: if jobs == nil { jobs = []Job{} } so it marshals to []
}

func TestHandler_Retry_ServiceError_Get(t *testing.T) {
	mockRepo := new(MockRepo)
	mockPub := new(MockPublisher)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	svc := job.NewService(mockRepo, mockPub, logger)
	handler := job.NewHandler(svc)

	jobID := "error-job"
	mockRepo.On("Get", mock.Anything, jobID).Return(nil, errors.New("db error"))

	req := httptest.NewRequest("POST", "/jobs/"+jobID+"/retry", nil)
	req.SetPathValue("id", jobID)
	w := httptest.NewRecorder()

	handler.Retry(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	mockRepo.AssertExpectations(t)
}

func TestHandler_Retry_ServiceError_Publish(t *testing.T) {
	mockRepo := new(MockRepo)
	mockPub := new(MockPublisher)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	svc := job.NewService(mockRepo, mockPub, logger)
	handler := job.NewHandler(svc)

	jobID := "publish-fail-job"
	j := &job.Job{
		ID:      jobID,
		Payload: []byte(`{"type": "web", "url": "http://example.com"}`),
	}

	mockRepo.On("Get", mock.Anything, jobID).Return(j, nil)
	mockPub.On("Publish", config.TopicIngestWeb, mock.Anything).Return(errors.New("nsq error"))

	req := httptest.NewRequest("POST", "/jobs/"+jobID+"/retry", nil)
	req.SetPathValue("id", jobID)
	w := httptest.NewRecorder()

	handler.Retry(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	mockRepo.AssertExpectations(t)
	mockPub.AssertExpectations(t)
}

func TestService_Retry_ContextCancellation(t *testing.T) {
	mockRepo := new(MockRepo)
	mockPub := new(MockPublisher)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	svc := job.NewService(mockRepo, mockPub, logger)

	jobID := "cancel-job"
	j := &job.Job{
		ID:      jobID,
		Payload: []byte(`{"type": "web"}`),
	}

	mockRepo.On("Get", mock.Anything, jobID).Return(j, nil)

	// We simulate a long running publish or just context cancellation
	// Since the service launches a goroutine for publish and selects on ctx.Done(),
	// we can try to block publish? But MockPublisher is synchronous.
	// Actually, the service code:
	// 	go func() { done <- s.pub.Publish(...) }()
	// 	select { case <-done: ... case <-ctx.Done(): ... }
	// So if we make Publish block, we can test cancellation.
	// But testify mock doesn't easily support blocking forever.
	// We can use .Run(func(args mock.Arguments) { time.Sleep(...) })

	mockPub.On("Publish", config.TopicIngestWeb, mock.Anything).Run(func(args mock.Arguments) {
		time.Sleep(100 * time.Millisecond)
	}).Return(nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := svc.Retry(ctx, jobID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestService_Retry_InvalidPayload(t *testing.T) {
	mockRepo := new(MockRepo)
	mockPub := new(MockPublisher)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	svc := job.NewService(mockRepo, mockPub, logger)

	jobID := "invalid-payload-job"
	j := &job.Job{
		ID:      jobID,
		Payload: []byte(`{invalid-json}`),
	}

	mockRepo.On("Get", mock.Anything, jobID).Return(j, nil)

	err := svc.Retry(context.Background(), jobID)
	assert.Error(t, err)
	// json unmarshal error
}

func TestService_Retry_DeleteError(t *testing.T) {
	mockRepo := new(MockRepo)
	mockPub := new(MockPublisher)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	svc := job.NewService(mockRepo, mockPub, logger)

	jobID := "delete-fail-job"
	j := &job.Job{
		ID:      jobID,
		Payload: []byte(`{"type": "web"}`),
	}

	mockRepo.On("Get", mock.Anything, jobID).Return(j, nil)
	mockPub.On("Publish", config.TopicIngestWeb, mock.Anything).Return(nil)
	mockRepo.On("Delete", mock.Anything, jobID).Return(errors.New("delete failed"))

	err := svc.Retry(context.Background(), jobID)
	assert.Error(t, err)
	assert.Equal(t, "delete failed", err.Error())
}
