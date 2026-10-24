package get

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
)

var (
	uuidPattern      = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	errInvalidCardID = errors.New("invalid card id")
)

type request struct {
	ID string
}

type response struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

func bindAndValidate(r *http.Request) (request, error) {
	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		return request{}, errInvalidCardID
	}
	return request{ID: id}, nil
}

func writeResponse(w http.ResponseWriter, card domain.Card) {
	resp := response{
		ID:          card.ID,
		Title:       card.Title,
		Description: card.Description,
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
	switch {
	case errors.Is(err, errInvalidCardID):
		slog.Error("invalid card id", "error", err)
		http.Error(w, errInvalidCardID.Error(), http.StatusBadRequest)
	case errors.Is(err, domain.ErrCardNotFound):
		http.Error(w, domain.ErrCardNotFound.Error(), http.StatusNotFound)
	default:
		slog.Error("failed to get card", "error", err)
		http.Error(w, "failed to get card", http.StatusInternalServerError)
	}
}

