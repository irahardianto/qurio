package app

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nsqio/go-nsq"
	"github.com/stretchr/testify/assert"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"qurio/apps/backend/internal/config"
)

func TestNew(t *testing.T) {
	// 1. Mock DB
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// 2. Mock Weaviate
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := weaviate.Config{
		Host:   server.URL[7:],
		Scheme: "http",
	}
	wClient, err := weaviate.NewClient(cfg)
	assert.NoError(t, err)

	// 3. Mock NSQ
	// NSQ Producer doesn't connect immediately?
	nsqCfg := nsq.NewConfig()
	producer, err := nsq.NewProducer("localhost:4150", nsqCfg)
	assert.NoError(t, err)

	// 4. Config
	appCfg := &config.Config{}

	// 5. Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Execute
	app, err := New(appCfg, db, wClient, producer, logger)
	assert.NoError(t, err)
	assert.NotNil(t, app)
	assert.NotNil(t, app.Handler)
	assert.NotNil(t, app.SourceService)
	assert.NotNil(t, app.ResultConsumer)

	// Verify Route (Integration-ish)
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	app.Handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
