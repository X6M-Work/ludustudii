package create

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
	"github.com/labstack/echo/v5"
)

type request struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type response struct {
	ID string `json:"id"`
}

type creator interface {
	Create(ctx context.Context, p createParams) (string, error)
}

type Handler struct {
	store    creator
	bind     func(c *echo.Context, i any) error
	sanitize func(req *request)
	validate func(req request) error
}

func NewHandler(s creator) *Handler {
	return &Handler{
		store:    s,
		bind:     bind,
		sanitize: sanitize,
		validate: validate,
	}
}

func (h *Handler) Handle(c *echo.Context) error {
	ctx := c.Request().Context()

	var req request
	if err := h.bind(c, &req); err != nil {
		slog.InfoContext(ctx, domain.TextBinding, "error", err)
		return c.JSON(http.StatusBadRequest, erresp.Binding())
	}

	h.sanitize(&req)

	if err := h.validate(req); err != nil {
		slog.InfoContext(ctx, domain.TextValidation, "error", err)
		return c.JSON(http.StatusBadRequest, erresp.Validation(err))
	}

	id, err := h.store.Create(ctx, toParams(req))
	if err != nil {
		slog.ErrorContext(ctx, domain.TextUnexpected, "error", err)
		return c.JSON(http.StatusInternalServerError, erresp.Unexpected())
	}

	return c.JSON(http.StatusCreated, toResp(id))
}

func bind(c *echo.Context, i any) error {
	return c.Bind(i)
}

func validate(req request) error {
	return errors.Join(
		domain.ValidateTitle(req.Title),
		domain.ValidateDescription(req.Description),
	)
}

func sanitize(req *request) {
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
}

func toParams(req request) createParams {
	return createParams{
		Title:       req.Title,
		Description: req.Description,
	}
}

func toResp(id string) response {
	return response{ID: id}
}
