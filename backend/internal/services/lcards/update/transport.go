package update

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
)

var (
	uuidPattern            = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	errInvalidCardID       = errors.New("invalid card id")
	errRequestBodyTooLarge = errors.New("request body too large")
	errInvalidRequestBody  = errors.New("failed to decode request body")
)

type request struct {
	ID          string `json:"-"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

func bindAndValidate(r *http.Request) (request, error) {
	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		return request{}, errInvalidCardID
	}

	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // Max 1MB

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return request{}, errors.Join(errRequestBodyTooLarge, err)
		}
		return request{}, errors.Join(errInvalidRequestBody, err)
	}

	req.ID = id
	return req, nil
}

func writeResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errInvalidCardID):
		slog.Error("invalid card id", "error", err)
		http.Error(w, errInvalidCardID.Error(), http.StatusBadRequest)
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
	case errors.Is(err, domain.ErrCardNotFound):
		http.Error(w, domain.ErrCardNotFound.Error(), http.StatusNotFound)
	default:
		slog.Error("failed to update card", "error", err)
		http.Error(w, "failed to update card", http.StatusInternalServerError)
	}
}

