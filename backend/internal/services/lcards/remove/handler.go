// Package remove implements deleting a learning card.
package remove

import (
	"context"
	"net/http"
	"strings"

	"github.com/apotourlyan/ludustudii/internal/kit/etag"
	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
)

type deleter interface {
	Delete(ctx context.Context, p deleteParams) error
}

// Handler serves the DELETE /api/lcards/{id} endpoint.
type Handler struct {
	store           deleter
	validateParamID func(id string) error
	parseVersion    func(string) (int64, error)
}

// NewHandler returns a Handler that uses s for storage.
func NewHandler(s deleter) *Handler {
	return &Handler{
		store:           s,
		validateParamID: validateParamID,
		parseVersion:    etag.ParseVersion,
	}
}

// Handle deletes the card whose ID is the {id} path parameter and responds 204,
// 400 for an invalid ID, 428 for missing or invalid If-Match, 404 for an
// unknown ID, or 412 for a stale version.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	id := strings.ToLower(r.PathValue("id"))
	if err := h.validateParamID(id); err != nil {
		return httpkit.Param(err)
	}

	version, err := h.parseVersion(strings.Join(r.Header.Values("If-Match"), ","))
	if err != nil {
		return httpkit.PreconditionRequired(err)
	}

	if err := h.store.Delete(ctx, deleteParams{ID: id, Version: version}); err != nil {
		return err
	}

	return httpkit.NoContent(w, http.StatusNoContent)
}

func validateParamID(id string) error {
	return validator.UUID7("id", id)
}
