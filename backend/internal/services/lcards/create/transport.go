package create

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
)

var (
	errRequestBodyTooLarge = errors.New("request body too large")
	errInvalidRequestBody  = errors.New("failed to decode request body")
)

type request struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type response struct {
	ID string `json:"id"`
}

func bindAndValidate(r *http.Request) (request, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // Max 1MB

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return request{}, errors.Join(errRequestBodyTooLarge, err)
		}
		return request{}, errors.Join(errInvalidRequestBody, err)
	}

	return req, nil
}

func writeResponse(w http.ResponseWriter, card domain.Card) {
	resp := response{ID: card.ID}
	data, err := json.Marshal(resp)
	if err != nil {
		slog.Error("failed to marshal response", "error", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(data); err != nil {
		slog.Error("failed to write response", "error", err)
	}
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errRequestBodyTooLarge):
		slog.Error("request body too large", "error", err)
		http.Error(w, errRequestBodyTooLarge.Error(), http.StatusRequestEntityTooLarge)
	case errors.Is(err, errInvalidRequestBody):
		slog.Error("failed to decode request body", "error", err)
		http.Error(w, errInvalidRequestBody.Error(), http.StatusBadRequest)
	case errors.Is(err, domain.ErrTitleRequired),
		errors.Is(err, domain.ErrTitleTooLong),
		errors.Is(err, domain.ErrDescriptionRequired),
		errors.Is(err, domain.ErrDescriptionTooLong):
		slog.Error("invalid request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		slog.Error("failed to create card", "error", err)
		http.Error(w, "failed to create card", http.StatusInternalServerError)
	}
}

