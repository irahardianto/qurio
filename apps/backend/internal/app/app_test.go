package app

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"qurio/apps/backend/internal/config"
)

func TestNew_Success(t *testing.T) {
	// Arrange
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockVec := &MockVectorStore{}
	mockPub := &MockTaskPublisher{}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := &config.Config{}

	// Act
	application, err := New(cfg, db, mockVec, mockPub, logger)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, application)
	assert.NotNil(t, application.Handler)
	assert.NotNil(t, application.SourceService)
	assert.NotNil(t, application.ResultConsumer)

	// Verify Routes
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/sources"},
		{"POST", "/sources"},
		{"GET", "/settings"},
		// {"GET", "/stats"}, // Requires DB mocks, skipping for simple connectivity check
		// {"GET", "/mcp/sse"}, // Blocking SSE, skipping
	}

	ts := httptest.NewServer(application.Handler)
	defer ts.Close()

	for _, rt := range routes {
		req, _ := http.NewRequest(rt.method, ts.URL+rt.path, nil)
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		if resp != nil {
			resp.Body.Close()
		}
		// Should NOT be 404. Might be 401, 500, 200 depending on mock, but 404 means route not found.
		assert.NotEqual(t, http.StatusNotFound, resp.StatusCode, "Route %s %s not found", rt.method, rt.path)
	}
}

type FakeDB struct{}

func (f *FakeDB) PingContext(ctx context.Context) error { return nil }
func (f *FakeDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row { return nil }
func (f *FakeDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) { return nil, nil }
func (f *FakeDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) { return nil, nil }
func (f *FakeDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) { return nil, nil }

func TestNew_PanicsOnInvalidDB(t *testing.T) {
	// Arrange
	mockVec := &MockVectorStore{}
	mockPub := &MockTaskPublisher{}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := &config.Config{}
	
	fakeDB := &FakeDB{}

	// Act & Assert
	assert.Panics(t, func() {
		_, _ = New(cfg, fakeDB, mockVec, mockPub, logger)
	})
}