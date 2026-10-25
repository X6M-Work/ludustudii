package update

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
	"github.com/labstack/echo/v5"
)

const (
	TextInvalidParamID  = "invalid param id"
	TextBindingError    = "body binding error"
	TextValidationError = "validation error"
	TextUnexpectedError = "unexpected error"
	TextNotFound        = "card not found"
)

type request struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type executor interface {
	Update(ctx context.Context, p updateParams) error
}

type Handler struct {
	store executor
}

func NewHandler(s executor) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Handle(c *echo.Context) error {
	ctx := c.Request().Context()

	id := c.Param("id")
	if err := validator.UUID7("id", id); err != nil {
		slog.ErrorContext(ctx, TextInvalidParamID, "error", err)
		return c.JSON(http.StatusBadRequest, erresp.New(TextInvalidParamID))
	}

	var req request
	if err := c.Bind(&req); err != nil {
		slog.ErrorContext(ctx, TextBindingError, "error", err)
		return c.JSON(http.StatusBadRequest, erresp.New(TextBindingError))
	}

	if err := validate(sanitize(&req)); err != nil {
		slog.ErrorContext(ctx, TextValidationError, "error", err)
		return c.JSON(http.StatusBadRequest, erresp.New(TextValidationError))
	}

	if err := h.store.Update(ctx, toParams(req, id)); err != nil {
		if errors.Is(err, ErrNotFound) {
			slog.ErrorContext(ctx, TextNotFound, "error", err)
			return c.JSON(http.StatusNotFound, erresp.New(TextNotFound))
		}
		slog.ErrorContext(ctx, TextUnexpectedError, "error", err)
		return c.JSON(http.StatusInternalServerError, erresp.New(TextUnexpectedError))
	}

	return c.NoContent(http.StatusNoContent)
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

func toParams(req request, id string) updateParams {
	return updateParams{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
	}
}
