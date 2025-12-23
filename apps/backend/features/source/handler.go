package source

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL        string   `json:"url"`
		MaxDepth   int      `json:"max_depth"`
		Exclusions []string `json:"exclusions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "VALIDATION_ERROR", err.Error(), http.StatusBadRequest)
		return
	}

	src := &Source{
		URL:        req.URL,
		MaxDepth:   req.MaxDepth,
		Exclusions: req.Exclusions,
	}
	if err := h.service.Create(r.Context(), src); err != nil {
		if err.Error() == "Duplicate detected" {
			h.writeError(w, "CONFLICT", err.Error(), http.StatusConflict)
			return
		}
		// Log the actual error for debugging
		slog.Error("operation failed", "error", err, "url", req.URL)
		h.writeError(w, "INTERNAL_ERROR", "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(src)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sources, err := h.service.List(r.Context())
	if err != nil {
		h.writeError(w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
		return
	}

	// Ensure we return [] instead of null for empty list
	if sources == nil {
		sources = []Source{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sources)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.service.Delete(r.Context(), id); err != nil {
		h.writeError(w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ReSync(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.service.ReSync(r.Context(), id); err != nil {
		h.writeError(w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	detail, err := h.service.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

func (h *Handler) writeError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
		"correlationId": uuid.New().String(),
	}

	json.NewEncoder(w).Encode(resp)
}
