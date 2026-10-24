package delete

import (
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

func bindAndValidate(r *http.Request) (request, error) {
	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		return request{}, errInvalidCardID
	}
	return request{ID: id}, nil
}

func writeResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errInvalidCardID):
		slog.Error("invalid card id", "error", err)
		http.Error(w, errInvalidCardID.Error(), http.StatusBadRequest)
	case errors.Is(err, domain.ErrCardNotFound):
		http.Error(w, domain.ErrCardNotFound.Error(), http.StatusNotFound)
	default:
		slog.Error("failed to delete card", "error", err)
		http.Error(w, "failed to delete card", http.StatusInternalServerError)
	}
}

