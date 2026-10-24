package list

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
)

type cardResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

func writeResponse(w http.ResponseWriter, cards []domain.Card) {
	// An empty collection must encode as [] rather than null.
	resp := make([]cardResponse, 0, len(cards))
	for _, card := range cards {
		resp = append(resp, cardResponse{
			ID:          card.ID,
			Title:       card.Title,
			Description: card.Description,
		})
	}

	data, err := json.Marshal(resp)
	if err != nil {
		slog.Error("failed to marshal response", "error", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		slog.Error("failed to write response", "error", err)
	}
}

func writeError(w http.ResponseWriter, err error) {
	slog.Error("failed to get cards", "error", err)
	http.Error(w, "failed to get cards", http.StatusInternalServerError)
}

