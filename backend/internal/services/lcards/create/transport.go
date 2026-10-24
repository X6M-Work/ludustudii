package create

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
	"github.com/labstack/echo/v5"
)

var (
	errRequestBodyTooLarge = errors.New("request body too large")
	errInvalidRequestBody  = errors.New("failed to decode request body")
)

type request struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type response struct {
	ID string `json:"id"`
}

func bindAndValidate(c *echo.Context) (request, error) {
	r := c.Request()
	r.Body = http.MaxBytesReader(c.Response(), r.Body, 1<<20) // Max 1MB

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return request{}, errors.Join(errRequestBodyTooLarge, err)
		}
		return request{}, errors.Join(errInvalidRequestBody, err)
	}

	return req, nil
}

func writeResponse(c *echo.Context, card domain.Card) error {
	resp := response{ID: card.ID}
	return c.JSON(http.StatusCreated, resp)
}

func writeError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, errRequestBodyTooLarge):
		slog.Error("request body too large", "error", err)
		return c.String(http.StatusRequestEntityTooLarge, errRequestBodyTooLarge.Error()+"\n")
	case errors.Is(err, errInvalidRequestBody):
		slog.Error("failed to decode request body", "error", err)
		return c.String(http.StatusBadRequest, errInvalidRequestBody.Error()+"\n")
	case errors.Is(err, domain.ErrTitleRequired),
		errors.Is(err, domain.ErrTitleTooLong),
		errors.Is(err, domain.ErrDescriptionRequired),
		errors.Is(err, domain.ErrDescriptionTooLong):
		slog.Error("invalid request", "error", err)
		return c.String(http.StatusBadRequest, err.Error()+"\n")
	default:
		slog.Error("failed to create card", "error", err)
		return c.String(http.StatusInternalServerError, "failed to create card\n")
	}
}
