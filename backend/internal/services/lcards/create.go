package lcards

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"
)

type createCardRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type createCardResponse struct {
	ID string `json:"id"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // Max 1MB

	var req createCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			slog.ErrorContext(r.Context(), "request body too large", "error", err)
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		slog.ErrorContext(r.Context(), "failed to decode request body", "error", err)
		http.Error(w, "failed to decode request body", http.StatusBadRequest)
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	if err := validateCreateCardRequest(req); err != nil {
		slog.ErrorContext(r.Context(), "invalid request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sql := `INSERT INTO learning_cards (title, description) VALUES ($1, $2) RETURNING id`
	var id string
	err := h.db.QueryRowContext(r.Context(), sql, req.Title, req.Description).Scan(&id)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to insert card", "error", err)
		http.Error(w, "failed to create card", http.StatusInternalServerError)
		return
	}

	resp := createCardResponse{ID: id}
	data, err := json.Marshal(resp)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to marshal response", "error", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(data); err != nil {
		slog.ErrorContext(r.Context(), "failed to write response", "error", err)
	}
}

func validateCreateCardRequest(req createCardRequest) error {
	if req.Title == "" {
		return fmt.Errorf("%q is required", "title")
	}
	if utf8.RuneCountInString(req.Title) > 200 {
		return fmt.Errorf("%q must be at most %d characters", "title", 200)
	}
	if req.Description == "" {
		return fmt.Errorf("%q is required", "description")
	}
	if utf8.RuneCountInString(req.Description) > 2000 {
		return fmt.Errorf("%q must be at most %d characters", "description", 2000)
	}
	return nil
}
