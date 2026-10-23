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

type updateCardRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		slog.ErrorContext(r.Context(), "invalid card id", "id", id)
		http.Error(w, "invalid card id", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // Max 1MB

	var req updateCardRequest
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
	if err := validateUpdateCardRequest(req); err != nil {
		slog.ErrorContext(r.Context(), "invalid request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `UPDATE learning_cards SET title = $1, description = $2 WHERE id = $3`
	res, err := h.db.ExecContext(r.Context(), query, req.Title, req.Description, id)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to update card", "error", err)
		http.Error(w, "failed to update card", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to check rows affected", "error", err)
		http.Error(w, "failed to update card", http.StatusInternalServerError)
		return
	}
	if rowsAffected == 0 {
		http.Error(w, "card not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func validateUpdateCardRequest(req updateCardRequest) error {
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

