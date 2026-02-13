package job

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"qurio/apps/backend/internal/middleware"
)

type Handler struct {
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	correlationID := middleware.GetCorrelationID(ctx)

	slog.InfoContext(ctx, "listing failed jobs", "correlationId", correlationID)

	jobs, err := h.service.List(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list jobs", "error", err, "correlationId", correlationID)
		h.writeError(ctx, w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
		return
	}

	if jobs == nil {
		jobs = []Job{}
	}

	w.Header().Set("Content-Type", "application/json")
	resp := map[string]interface{}{
		"data": jobs,
		"meta": map[string]int{"count": len(jobs)},
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.ErrorContext(ctx, "failed to encode response", "error", err)
	}
}

func (h *Handler) Retry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	correlationID := middleware.GetCorrelationID(ctx)
	id := r.PathValue("id")

	slog.InfoContext(ctx, "retrying job", "id", id, "correlationId", correlationID)

	if err := h.service.Retry(ctx, id); err != nil {
		slog.ErrorContext(ctx, "failed to retry job", "id", id, "error", err, "correlationId", correlationID)
		if errors.Is(err, sql.ErrNoRows) {
			h.writeError(ctx, w, "NOT_FOUND", "Job not found", http.StatusNotFound)
			return
		}
		h.writeError(ctx, w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"data": "job retried"}); err != nil {
		slog.ErrorContext(ctx, "failed to encode response", "error", err)
	}
}

func (h *Handler) writeError(ctx context.Context, w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
		"correlationId": middleware.GetCorrelationID(ctx),
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}
