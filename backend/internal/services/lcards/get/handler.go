// Package get implements fetching a single learning card.
package get

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
	"github.com/apotourlyan/ludustudii/internal/kit/etag"
	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
)

type response struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     int64  `json:"version"`
}

type getter interface {
	Get(ctx context.Context, id string) (row, error)
}

// Handler serves the GET /api/lcards/:id endpoint.
type Handler struct {
	store           getter
	validateParamID func(id string) error
}

// NewHandler returns a Handler that uses s for storage.
func NewHandler(s getter) *Handler {
	return &Handler{store: s, validateParamID: validateParamID}
}

// Handle responds 200 with the card whose ID is the :id path parameter, 400 if
// the ID is not a UUIDv7 (case-insensitive) and 404 if no card has it.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	id := strings.ToLower(r.PathValue("id"))
	if err := h.validateParamID(id); err != nil {
		slog.DebugContext(ctx, domain.TextInvalidID, "error", err)
		return httpkit.JSON(w, http.StatusBadRequest, erresp.Param())
	}

	data, err := h.store.Get(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			slog.DebugContext(ctx, domain.TextNotFound, "error", err)
			return httpkit.JSON(w, http.StatusNotFound, erresp.NotFound())
		}
		slog.ErrorContext(ctx, domain.TextUnexpected, "error", err)
		return httpkit.JSON(w, http.StatusInternalServerError, erresp.Unexpected())
	}

	w.Header().Set("ETag", etag.Format(data.Version))
	return httpkit.JSON(w, http.StatusOK, toResp(data))
}

func validateParamID(id string) error {
	return validator.UUID7("id", id)
}

func toResp(data row) response {
	return response(data)
}
