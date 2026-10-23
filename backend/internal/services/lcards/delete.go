package lcards

import (
	"log/slog"
	"net/http"
)

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		slog.ErrorContext(r.Context(), "invalid card id", "id", id)
		http.Error(w, "invalid card id", http.StatusBadRequest)
		return
	}

	query := `DELETE FROM learning_cards WHERE id = $1`
	res, err := h.db.ExecContext(r.Context(), query, id)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to delete card", "error", err)
		http.Error(w, "failed to delete card", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to check rows affected", "error", err)
		http.Error(w, "failed to delete card", http.StatusInternalServerError)
		return
	}
	if rowsAffected == 0 {
		http.Error(w, "card not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
