package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCorrelationID_Middleware(t *testing.T) {
	tests := []struct {
		name           string
		incomingHeader string
		expectHeader   bool
		expectSameID   bool
	}{
		{
			name:           "Should Generate ID When Missing",
			incomingHeader: "",
			expectHeader:   true,
			expectSameID:   false,
		},
		{
			name:           "Should Preserve Existing ID",
			incomingHeader: "test-correlation-id-123",
			expectHeader:   true,
			expectSameID:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.incomingHeader != "" {
				req.Header.Set("X-Correlation-ID", tt.incomingHeader)
			}
			rec := httptest.NewRecorder()

			handler := CorrelationID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				id := GetCorrelationID(r.Context())
				if tt.expectHeader {
					assert.NotEmpty(t, id)
				}
				if tt.expectSameID {
					assert.Equal(t, tt.incomingHeader, id)
				}
			}))

			handler.ServeHTTP(rec, req)

			// Check Response Header
			respHeader := rec.Header().Get("X-Correlation-ID")
			if tt.expectHeader {
				assert.NotEmpty(t, respHeader)
			}
			if tt.expectSameID {
				assert.Equal(t, tt.incomingHeader, respHeader)
			}
		})
	}
}

func TestGetCorrelationID_Extraction(t *testing.T) {
	// Test extraction from empty context
	req := httptest.NewRequest("GET", "/", nil)
	id := GetCorrelationID(req.Context())
	assert.Equal(t, "unknown", id)

	// Test extraction from context with ID
	ctx := WithCorrelationID(req.Context(), "test-id")
	id = GetCorrelationID(ctx)
	assert.Equal(t, "test-id", id)
}
