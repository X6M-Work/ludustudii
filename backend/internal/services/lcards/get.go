package lcards

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type getCardResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		slog.ErrorContext(r.Context(), "invalid card id", "id", id)
		http.Error(w, "invalid card id", http.StatusBadRequest)
		return
	}

	query := `SELECT id, title, description FROM learning_cards WHERE id = $1`
	var resp getCardResponse
	err := h.db.QueryRowContext(r.Context(), query, id).Scan(&resp.ID, &resp.Title, &resp.Description)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "card not found", http.StatusNotFound)
			return
		}
		slog.ErrorContext(r.Context(), "failed to query card", "error", err)
		http.Error(w, "failed to get card", http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(resp)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to marshal response", "error", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		slog.ErrorContext(r.Context(), "failed to write response", "error", err)
	}
}
