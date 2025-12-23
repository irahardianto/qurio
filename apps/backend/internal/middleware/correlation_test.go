package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCorrelationID(t *testing.T) {
	handler := CorrelationID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := r.Context().Value(CorrelationKey).(string)
		if !ok || id == "" {
			t.Error("correlation id missing from context")
		}
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("X-Correlation-ID") == "" {
		t.Error("header missing")
	}
}
