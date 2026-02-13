package stats

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"qurio/apps/backend/internal/middleware"
)

type SourceRepo interface {
	Count(ctx context.Context) (int, error)
}

type JobRepo interface {
	Count(ctx context.Context) (int, error)
}

type VectorStore interface {
	CountChunks(ctx context.Context) (int, error)
}

type Handler struct {
	sourceRepo  SourceRepo
	jobRepo     JobRepo
	vectorStore VectorStore
}

func NewHandler(s SourceRepo, j JobRepo, v VectorStore) *Handler {
	return &Handler{sourceRepo: s, jobRepo: j, vectorStore: v}
}

type StatsResponse struct {
	Sources    int `json:"sources"`
	Documents  int `json:"documents"`
	FailedJobs int `json:"failed_jobs"`
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	correlationID := middleware.GetCorrelationID(ctx)

	slog.InfoContext(ctx, "getting stats", "correlationId", correlationID)

	sCount, err := h.sourceRepo.Count(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to count sources", "error", err, "correlationId", correlationID)
		h.writeError(ctx, w, "INTERNAL_ERROR", "failed to count sources", http.StatusInternalServerError)
		return
	}

	jCount, err := h.jobRepo.Count(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to count jobs", "error", err, "correlationId", correlationID)
		h.writeError(ctx, w, "INTERNAL_ERROR", "failed to count jobs", http.StatusInternalServerError)
		return
	}

	dCount, err := h.vectorStore.CountChunks(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to count documents", "error", err, "correlationId", correlationID)
		h.writeError(ctx, w, "INTERNAL_ERROR", "failed to count documents", http.StatusInternalServerError)
		return
	}

	resp := StatsResponse{
		Sources:    sCount,
		Documents:  dCount,
		FailedJobs: jCount,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"data": resp}); err != nil {
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
