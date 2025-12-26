package stats

import (
	"context"
	"encoding/json"
	"net/http"
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
	
	sCount, err := h.sourceRepo.Count(ctx)
	if err != nil {
		http.Error(w, "failed to count sources", http.StatusInternalServerError)
		return
	}

	jCount, err := h.jobRepo.Count(ctx)
	if err != nil {
		http.Error(w, "failed to count jobs", http.StatusInternalServerError)
		return
	}

	dCount, err := h.vectorStore.CountChunks(ctx)
	if err != nil {
		http.Error(w, "failed to count documents", http.StatusInternalServerError)
		return
	}

	resp := StatsResponse{
		Sources:    sCount,
		Documents:  dCount,
		FailedJobs: jCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
