// Package create implements creating a learning card.
package create

import (
	"context"
	"errors"
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/kit/etag"
	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
	"github.com/apotourlyan/ludustudii/internal/kit/sanitizer"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
)

type request struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type response struct {
	ID string `json:"id"`
}

type creator interface {
	Create(ctx context.Context, p createParams) (row, error)
}

// Handler serves the POST /api/lcards endpoint.
type Handler struct {
	store    creator
	bind     func(r *http.Request, i any) error
	validate func(req request) error
}

// NewHandler returns a Handler that uses s for storage.
func NewHandler(s creator) *Handler {
	return &Handler{
		store:    s,
		bind:     bind,
		validate: validate,
	}
}

// Handle sanitizes and validates the JSON title and description, inserts the
// card and responds 201 with its ID in the body and its URL in the Location
// header. Invalid input gets 400. An oversized body (413) or an unsupported
// Content-Type (415) is returned for the router's error handler to report.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	var req request
	if err := h.bind(r, &req); err != nil {
		return err
	}

	sanitize(&req)

	if err := h.validate(req); err != nil {
		return httpkit.Validation(err)
	}

	data, err := h.store.Create(ctx, toParams(req))
	if err != nil {
		return err
	}

	w.Header().Set("ETag", etag.Format(data.Version))
	w.Header().Set("Location", domain.CardsPath+"/"+data.ID)
	return httpkit.JSON(w, http.StatusCreated, toResp(data))
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

func sanitize(req *request) {
	req.Title = sanitizer.String(req.Title)
	req.Description = sanitizer.String(req.Description)
}

func toParams(req request) createParams {
	return createParams(req)
}

func toResp(data row) response {
	return response{ID: data.ID}
}
