package lcards

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	query := `SELECT id, title, description FROM learning_cards ORDER BY id DESC`
	rows, err := h.db.QueryContext(r.Context(), query)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to query cards", "error", err)
		http.Error(w, "failed to get cards", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// An empty collection must encode as [] rather than null.
	cards := make([]getCardResponse, 0)
	for rows.Next() {
		var card getCardResponse
		if err := rows.Scan(&card.ID, &card.Title, &card.Description); err != nil {
			slog.ErrorContext(r.Context(), "failed to scan card", "error", err)
			http.Error(w, "failed to get cards", http.StatusInternalServerError)
			return
		}
		cards = append(cards, card)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(r.Context(), "failed to iterate cards", "error", err)
		http.Error(w, "failed to get cards", http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(cards)
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
