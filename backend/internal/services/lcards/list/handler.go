// Package list implements listing learning cards with keyset pagination.
package list

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
	"github.com/labstack/echo/v5"
)

type request struct {
	Limit  string `query:"limit"`
	Cursor string `query:"cursor"`
}

type item struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
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
	bind     func(c *echo.Context, i any) error
	validate func(req request) error
}

// NewHandler returns a Handler that uses s for storage.
func NewHandler(s lister) *Handler {
	return &Handler{
		store:    s,
		bind:     bind,
		validate: validate,
	}
}

// Handle responds 200 with a page of cards ordered newest first. The optional
// limit query parameter sets the page size (domain.ListMinLimit to
// domain.ListMaxLimit, default domain.ListDefaultLimit). The optional cursor
// is the next_cursor of the previous page, matched case-insensitively;
// next_cursor is null on the last page. Invalid parameters get 400.
func (h *Handler) Handle(c *echo.Context) error {
	ctx := c.Request().Context()

	var req request
	if err := h.bind(c, &req); err != nil {
		slog.DebugContext(ctx, domain.TextBinding, "error", err)
		return c.JSON(http.StatusBadRequest, erresp.Binding())
	}

	sanitize(&req)

	if err := h.validate(req); err != nil {
		slog.DebugContext(ctx, domain.TextValidation, "error", err)
		return c.JSON(http.StatusBadRequest, erresp.Validation(err))
	}

	data, err := h.store.List(ctx, toParams(req))
	if err != nil {
		slog.ErrorContext(ctx, domain.TextUnexpected, "error", err)
		return c.JSON(http.StatusInternalServerError, erresp.Unexpected())
	}

	return c.JSON(http.StatusOK, toResp(data))
}

// The Echo-wide binder only binds JSON bodies, so query params are bound here.
func bind(c *echo.Context, i any) error {
	return echo.BindQueryParams(c, i)
}

func validate(req request) error {
	return errors.Join(
		validateLimit(req.Limit),
		validateCursor(req.Cursor),
	)
}

func validateLimit(val string) error {
	if val == "" {
		return nil
	}
	if err := validator.Integer("limit", val); err != nil {
		return err
	}
	n, _ := strconv.Atoi(val)
	return validator.IntRange("limit", n, domain.ListMinLimit, domain.ListMaxLimit)
}

func validateCursor(val string) error {
	if val == "" {
		return nil
	}
	return validator.UUID7("cursor", val)
}

// sanitize lowercases the cursor, so it is accepted in any case like the
// card ID path parameter.
func sanitize(req *request) {
	req.Cursor = strings.ToLower(req.Cursor)
}

// toParams runs after validate, so Limit is either empty or a valid integer.
func toParams(req request) listParams {
	limit := domain.ListDefaultLimit
	if req.Limit != "" {
		limit, _ = strconv.Atoi(req.Limit)
	}
	return listParams{
		Limit:  limit,
		Cursor: req.Cursor,
	}
}

func toResp(data page) response {
	items := make([]item, 0, len(data.Rows))
	for _, d := range data.Rows {
		items = append(items, item{
			ID:          d.ID,
			Title:       d.Title,
			Description: d.Description,
		})
	}

	var next *string
	if data.NextCursor != "" {
		next = &data.NextCursor
	}

	return response{Items: items, NextCursor: next}
}
