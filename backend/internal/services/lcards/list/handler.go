package list

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

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

type Handler struct {
	store    lister
	bind     func(c *echo.Context, i any) error
	validate func(req request) error
}

func NewHandler(s lister) *Handler {
	return &Handler{
		store:    s,
		bind:     bind,
		validate: validate,
	}
}

func (h *Handler) Handle(c *echo.Context) error {
	ctx := c.Request().Context()

	var req request
	if err := h.bind(c, &req); err != nil {
		slog.DebugContext(ctx, domain.TextBinding, "error", err)
		return c.JSON(http.StatusBadRequest, erresp.Binding())
	}

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
