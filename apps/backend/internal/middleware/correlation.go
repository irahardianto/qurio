package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type key int

const CorrelationKey key = 0

func CorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Correlation-ID")
		if id == "" {
			id = uuid.New().String()
		}

		ctx := context.WithValue(r.Context(), CorrelationKey, id)
		w.Header().Set("X-Correlation-ID", id)

		slog.Info("request received", "method", r.Method, "path", r.URL.Path, "correlation_id", id) // #nosec G706 -- r.URL.Path is parsed by Go's net/http
		start := time.Now()

		next.ServeHTTP(w, r.WithContext(ctx))

		slog.Info("request completed", "method", r.Method, "path", r.URL.Path, "correlation_id", id, "duration", time.Since(start)) // #nosec G706
	})
}

func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(CorrelationKey).(string); ok {
		return id
	}
	return "unknown"
}

func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, CorrelationKey, id)
}
