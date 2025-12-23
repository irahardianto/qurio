package settings

import (
	"context"
	"encoding/json"
	"net/http"

	"qurio/apps/backend/internal/middleware"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	s, err := h.svc.Get(r.Context())
	if err != nil {
		h.writeError(r.Context(), w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"data": s})
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var s Settings
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		h.writeError(r.Context(), w, "VALIDATION_ERROR", err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.svc.Update(r.Context(), &s); err != nil {
		h.writeError(r.Context(), w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
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

	json.NewEncoder(w).Encode(resp)
}