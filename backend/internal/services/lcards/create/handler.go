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

const (
	TextBindingError    = "body binding error"
	TextValidationError = "validation error"
	TextUnexpectedError = "unexpected error"
)

type request struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type response struct {
	ID string `json:"id"`
}

type executor interface {
	Create(ctx context.Context, p createParams) (string, error)
}

type Handler struct {
	store executor
}

func NewHandler(s executor) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Handle(c *echo.Context) error {
	ctx := c.Request().Context()

	var req request
	if err := c.Bind(&req); err != nil {
		slog.ErrorContext(ctx, TextBindingError, "error", err)
		return c.JSON(http.StatusBadRequest, erresp.New(TextBindingError))
	}

	if err := validate(sanitize(&req)); err != nil {
		slog.ErrorContext(ctx, TextValidationError, "error", err)
		return c.JSON(http.StatusBadRequest, erresp.New(TextValidationError))
	}

	id, err := h.store.Create(ctx, toParams(req))
	if err != nil {
		slog.ErrorContext(ctx, TextUnexpectedError, "error", err)
		return c.JSON(http.StatusInternalServerError, erresp.New(TextUnexpectedError))
	}

	return c.JSON(http.StatusCreated, toResp(id))
}

func validate(req request) error {
	return errors.Join(
		domain.ValidateTitle(req.Title),
		domain.ValidateDescription(req.Description),
	)
}

func sanitize(req *request) request {
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	return *req
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
