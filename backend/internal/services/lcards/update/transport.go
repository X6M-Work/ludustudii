package update

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
	"github.com/labstack/echo/v5"
)

var (
	uuidPattern            = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	errInvalidCardID       = errors.New("invalid card id")
	errRequestBodyTooLarge = errors.New("request body too large")
	errInvalidRequestBody  = errors.New("failed to decode request body")
)

type request struct {
	ID          string `json:"-"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

func bindAndValidate(c *echo.Context) (request, error) {
	id := c.Param("id")
	if !uuidPattern.MatchString(id) {
		return request{}, errInvalidCardID
	}

	r := c.Request()
	r.Body = http.MaxBytesReader(c.Response(), r.Body, 1<<20) // Max 1MB

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return request{}, errors.Join(errRequestBodyTooLarge, err)
		}
		return request{}, errors.Join(errInvalidRequestBody, err)
	}

	req.ID = id
	return req, nil
}

func writeResponse(c *echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func writeError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, errInvalidCardID):
		slog.Error("invalid card id", "error", err)
		return c.String(http.StatusBadRequest, errInvalidCardID.Error()+"\n")
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
	case errors.Is(err, domain.ErrCardNotFound):
		return c.String(http.StatusNotFound, domain.ErrCardNotFound.Error()+"\n")
	default:
		slog.Error("failed to update card", "error", err)
		return c.String(http.StatusInternalServerError, "failed to update card\n")
	}
}
