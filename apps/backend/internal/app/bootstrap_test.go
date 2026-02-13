package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"qurio/apps/backend/internal/app"
	"qurio/apps/backend/internal/config"
)

func TestEnsureSchemaWithRetry_Success(t *testing.T) {
	mockStore := &app.MockVectorStore{
		EnsureSchemaErr: nil,
	}
	err := app.EnsureSchemaWithRetry(context.Background(), mockStore, 1, 1*time.Millisecond)
	assert.NoError(t, err)
}

type statefulMockStore struct {
	app.MockVectorStore
	callCount int
	failUntil int
}

func (m *statefulMockStore) EnsureSchema(ctx context.Context) error {
	m.callCount++
	if m.callCount <= m.failUntil {
		return errors.New("schema error")
	}
	return nil
}

func TestEnsureSchemaWithRetry_Retries(t *testing.T) {
	mock := &statefulMockStore{failUntil: 2}
	err := app.EnsureSchemaWithRetry(context.Background(), mock, 5, 1*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, 3, mock.callCount)
}

func TestEnsureSchemaWithRetry_Fail(t *testing.T) {
	mockStore := &app.MockVectorStore{
		EnsureSchemaErr: errors.New("permanent error"),
	}
	err := app.EnsureSchemaWithRetry(context.Background(), mockStore, 3, 1*time.Millisecond)
	assert.Error(t, err)
}

func TestBootstrap_ConfigurationError(t *testing.T) {
	cfg := &config.Config{
		DBHost: "invalid-host",
	}
	deps, err := app.Bootstrap(context.Background(), cfg)
	assert.Error(t, err)
	assert.Nil(t, deps)
}
