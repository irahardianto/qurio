package job

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type EventPublisher interface {
	Publish(topic string, body []byte) error
}

type Service struct {
	repo   Repository
	pub    EventPublisher
	logger *slog.Logger
}

func NewService(repo Repository, pub EventPublisher, logger *slog.Logger) *Service {
	return &Service{repo: repo, pub: pub, logger: logger}
}

func (s *Service) List(ctx context.Context) ([]Job, error) {
	return s.repo.List(ctx)
}

func (s *Service) Retry(ctx context.Context, id string) error {
	s.logger.Info("job retry started", "job_id", id)

	// 1. Get Job
	job, err := s.repo.Get(ctx, id)
	if err != nil {
		s.logger.Error("failed to get job", "job_id", id, "error", err)
		return err
	}

	// 2. Publish to NSQ with timeout
	done := make(chan error, 1)
	go func() {
		done <- s.pub.Publish("ingest.task", job.Payload)
	}()

	select {
	case err := <-done:
		if err != nil {
			s.logger.Error("failed to publish job", "job_id", id, "error", err)
			return err
		}
	case <-time.After(5 * time.Second):
		s.logger.Error("timeout waiting for NSQ publish", "job_id", id)
		return fmt.Errorf("timeout waiting for NSQ publish")
	case <-ctx.Done():
		return ctx.Err()
	}

	// 3. Delete Job
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete job", "job_id", id, "error", err)
		return err
	}

	s.logger.Info("job retry successful", "job_id", id)
	return nil
}

func (s *Service) Count(ctx context.Context) (int, error) {
	return s.repo.Count(ctx)
}

func (s *Service) ResetStuckJobs(ctx context.Context) (int64, error) {
	return 0, nil
}
