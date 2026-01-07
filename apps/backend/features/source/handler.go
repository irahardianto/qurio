package source

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"qurio/apps/backend/internal/middleware"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type       string   `json:"type"`
		URL        string   `json:"url"`
		MaxDepth   int      `json:"max_depth"`
		Exclusions []string `json:"exclusions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, "VALIDATION_ERROR", err.Error(), http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		h.writeError(r.Context(), w, "VALIDATION_ERROR", "URL is required", http.StatusBadRequest)
		return
	}

	src := &Source{
		Type:       req.Type,
		URL:        req.URL,
		MaxDepth:   req.MaxDepth,
		Exclusions: req.Exclusions,
	}
	if err := h.service.Create(r.Context(), src); err != nil {
		if err.Error() == "Duplicate detected" {
			h.writeError(r.Context(), w, "CONFLICT", err.Error(), http.StatusConflict)
			return
		}
		// Log the actual error for debugging
		slog.Error("operation failed", "error", err, "url", req.URL)
		h.writeError(r.Context(), w, "INTERNAL_ERROR", "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"data": src})
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	// 50 MB limit (enforced at reader level)
	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)

	// 50 MB limit (memory)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		h.writeError(r.Context(), w, "BAD_REQUEST", "File too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.writeError(r.Context(), w, "BAD_REQUEST", "Unable to retrieve file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate File Extension/MIME
	ext := filepath.Ext(header.Filename)
	validExts := map[string]bool{
		".pdf": true, ".md": true, ".txt": true, ".json": true, ".csv": true,
	}
	if !validExts[ext] {
		h.writeError(r.Context(), w, "BAD_REQUEST", "Unsupported file type", http.StatusBadRequest)
		return
	}

	// Create uploads directory if not exists
	uploadDir := os.Getenv("QURIO_UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		slog.Error("failed to create upload directory", "error", err, "path", uploadDir)
		h.writeError(r.Context(), w, "INTERNAL_ERROR", "Failed to create upload directory", http.StatusInternalServerError)
		return
	}

	// Generate filename
	filename := fmt.Sprintf("%s_%s", uuid.New().String(), filepath.Base(header.Filename))
	path := filepath.Join(uploadDir, filename)

	// Create file
	dst, err := os.Create(path)
	if err != nil {
		slog.Error("failed to create file", "error", err, "path", path)
		h.writeError(r.Context(), w, "INTERNAL_ERROR", "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Calculate hash while copying
	hash := sha256.New()
	mw := io.MultiWriter(dst, hash)

	if _, err := io.Copy(mw, file); err != nil {
		h.writeError(r.Context(), w, "INTERNAL_ERROR", "Failed to write file", http.StatusInternalServerError)
		return
	}

	fileHash := fmt.Sprintf("%x", hash.Sum(nil))

	// Call Service
	src, err := h.service.Upload(r.Context(), path, fileHash)
	if err != nil {
		// Clean up file if duplicate or error
		os.Remove(path)

		if err.Error() == "Duplicate detected" {
			h.writeError(r.Context(), w, "CONFLICT", err.Error(), http.StatusConflict)
			return
		}
		h.writeError(r.Context(), w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"data": src})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sources, err := h.service.List(r.Context())
	if err != nil {
		h.writeError(r.Context(), w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
		return
	}

	// Ensure we return [] instead of null for empty list
	if sources == nil {
		sources = []Source{}
	}

	w.Header().Set("Content-Type", "application/json")
	resp := map[string]interface{}{
		"data": sources,
		"meta": map[string]int{"count": len(sources)},
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.writeError(r.Context(), w, "NOT_FOUND", "Source not found", http.StatusNotFound)
			return
		}
		h.writeError(r.Context(), w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ReSync(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.service.ReSync(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.writeError(r.Context(), w, "NOT_FOUND", "Source not found", http.StatusNotFound)
			return
		}
		h.writeError(r.Context(), w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	detail, err := h.service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.writeError(r.Context(), w, "NOT_FOUND", "Source not found", http.StatusNotFound)
			return
		}
		h.writeError(r.Context(), w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": detail})
}

func (h *Handler) GetPages(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	pages, err := h.service.GetPages(r.Context(), id)
	if err != nil {
		h.writeError(r.Context(), w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
		return
	}
	if pages == nil {
		pages = []SourcePage{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": pages,
		"meta": map[string]int{"count": len(pages)},
	})
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
