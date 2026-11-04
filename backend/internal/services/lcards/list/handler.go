// Package list implements listing learning cards with keyset pagination.
package list

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
)

type request struct {
	Limit  int
	Cursor string
}

type item struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     int64  `json:"version"`
}

type response struct {
	Items      []item  `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

type lister interface {
	List(ctx context.Context, p listParams) (page, error)
}

// Handler serves the GET /api/lcards endpoint.
type Handler struct {
	store    lister
	parse    func(r *http.Request) (request, error)
	validate func(req request) error
}

// NewHandler returns a Handler that uses s for storage.
func NewHandler(s lister) *Handler {
	return &Handler{
		store:    s,
		parse:    parse,
		validate: validate,
	}
}

// Handle responds 200 with a page of cards ordered newest first. The optional
// limit query parameter sets the page size (domain.ListMinLimit to
// domain.ListMaxLimit, default domain.ListDefaultLimit). The optional cursor
// is the next_cursor of the previous page, matched case-insensitively;
// next_cursor is null on the last page. Invalid parameters get 400.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	req, err := h.parse(r)
	if err != nil {
		return err
	}

	sanitize(&req)

	if err := h.validate(req); err != nil {
		return httpkit.Validation(err)
	}

	data, err := h.store.List(ctx, toParams(req))
	if err != nil {
		return err
	}

	return httpkit.JSON(w, http.StatusOK, toResp(data))
}

func parse(r *http.Request) (request, error) {
	limit, err := httpkit.BindQueryInt(r, "limit", domain.ListDefaultLimit)
	if err != nil {
		return request{}, err
	}
	cursor := r.URL.Query().Get("cursor")
	return request{Limit: limit, Cursor: cursor}, nil
}

func validate(req request) error {
	return errors.Join(
		validator.IntRange("limit", req.Limit, domain.ListMinLimit, domain.ListMaxLimit),
		validator.EmptyOrUUID7("cursor", req.Cursor),
	)
}

// sanitize lowercases the cursor, so it is accepted in any case like the
// card ID path parameter.
func sanitize(req *request) {
	req.Cursor = strings.ToLower(req.Cursor)
}

func toParams(req request) listParams {
	return listParams(req)
}

func toResp(data page) response {
	items := make([]item, 0, len(data.Rows))
	for _, d := range data.Rows {
		items = append(items, item(d))
	}

	var next *string
	if data.NextCursor != "" {
		next = &data.NextCursor
	}

	return response{Items: items, NextCursor: next}
}
