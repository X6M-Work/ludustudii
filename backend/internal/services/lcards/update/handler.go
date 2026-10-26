package update

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
	"github.com/apotourlyan/ludustudii/internal/kit/sanitizer"
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
	"github.com/labstack/echo/v5"
)

type request struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type updater interface {
	Update(ctx context.Context, p updateParams) error
}

type Handler struct {
	store           updater
	bind            func(c *echo.Context, i any) error
	validate        func(req request) error
	validateParamID func(id string) error
}

func NewHandler(s updater) *Handler {
	return &Handler{
		store:           s,
		bind:            bind,
		validate:        validate,
		validateParamID: validateParamID,
	}
}

func (h *Handler) Handle(c *echo.Context) error {
	ctx := c.Request().Context()

	id := strings.ToLower(c.Param("id"))
	if err := h.validateParamID(id); err != nil {
		slog.InfoContext(ctx, domain.TextInvalidID, "error", err)
		return c.JSON(http.StatusBadRequest, erresp.Param())
	}

	var req request
	if err := h.bind(c, &req); err != nil {
		slog.InfoContext(ctx, domain.TextBinding, "error", err)
		return c.JSON(http.StatusBadRequest, erresp.Binding())
	}

	sanitize(&req)

	if err := h.validate(req); err != nil {
		slog.InfoContext(ctx, domain.TextValidation, "error", err)
		return c.JSON(http.StatusBadRequest, erresp.Validation(err))
	}

	if err := h.store.Update(ctx, toParams(req, id)); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			slog.InfoContext(ctx, domain.TextNotFound, "error", err)
			return c.JSON(http.StatusNotFound, erresp.NotFound())
		}
		slog.ErrorContext(ctx, domain.TextUnexpected, "error", err)
		return c.JSON(http.StatusInternalServerError, erresp.Unexpected())
	}

	return c.NoContent(http.StatusNoContent)
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

func validateParamID(id string) error {
	return validator.UUID7("id", id)
}

func sanitize(req *request) {
	req.Title = sanitizer.String(req.Title)
	req.Description = sanitizer.String(req.Description)
}

func toParams(req request, id string) updateParams {
	return updateParams{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
	}
}
