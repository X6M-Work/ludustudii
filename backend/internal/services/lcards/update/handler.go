// Package update implements replacing a learning card's content.
package update

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/apotourlyan/ludustudii/internal/kit/etag"
	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
	"github.com/apotourlyan/ludustudii/internal/kit/sanitizer"
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
)

type request struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type updater interface {
	Update(ctx context.Context, p updateParams) (int64, error)
}

// Handler serves the PUT /api/lcards/{id} endpoint.
type Handler struct {
	store           updater
	bind            func(r *http.Request, i any) error
	validate        func(req request) error
	validateParamID func(id string) error
	parseVersion    func(string) (int64, error)
}

// NewHandler returns a Handler that uses s for storage.
func NewHandler(s updater) *Handler {
	return &Handler{
		store:           s,
		bind:            bind,
		validate:        validate,
		parseVersion:    etag.ParseVersion,
		validateParamID: validateParamID,
	}
}

// Handle sanitizes and validates the JSON title and description, replaces
// them on the card whose ID is the {id} path parameter and responds 204.
// Invalid input or an ID that is not a UUIDv7 (case-insensitive) gets 400,
// an unknown ID gets 404, missing or invalid If-Match gets 428, and a stale
// version gets 412. Success includes the new version as an ETag. An oversized
// body (413) or an unsupported Content-Type (415) is returned for the router's
// error handler to report.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	id := strings.ToLower(r.PathValue("id"))
	if err := h.validateParamID(id); err != nil {
		return httpkit.Validation(err)
	}

	version, err := h.parseVersion(strings.Join(r.Header.Values("If-Match"), ","))
	if err != nil {
		return httpkit.PreconditionRequired(err)
	}

	var req request
	if err := h.bind(r, &req); err != nil {
		return err
	}

	sanitize(&req)

	if err := h.validate(req); err != nil {
		return httpkit.Validation(err)
	}

	newVersion, err := h.store.Update(ctx, toParams(req, id, version))
	if err != nil {
		return err
	}

	w.Header().Set("ETag", etag.Format(newVersion))
	return httpkit.NoContent(w, http.StatusNoContent)
}

func bind(r *http.Request, i any) error {
	return httpkit.BindJSON(r, i)
}

func validate(req request) error {
	return errors.Join(
		domain.ValidateTitle(req.Title),
		domain.ValidateDescription(req.Description),
	)
}

func validateParamID(id string) error {
	return validator.UUID7("id", id)
}

func sanitize(req *request) {
	req.Title = sanitizer.String(req.Title)
	req.Description = sanitizer.String(req.Description)
}

func toParams(req request, id string, version int64) updateParams {
	return updateParams{
		Version:     version,
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
	}
}
