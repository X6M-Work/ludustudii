package get

import (
	"errors"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
	"github.com/labstack/echo/v5"
)

var (
	uuidPattern      = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	errInvalidCardID = errors.New("invalid card id")
)

type request struct {
	ID string
}

type response struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

func bindAndValidate(c *echo.Context) (request, error) {
	id := c.Param("id")
	if !uuidPattern.MatchString(id) {
		return request{}, errInvalidCardID
	}
	return request{ID: id}, nil
}

func writeResponse(c *echo.Context, card domain.Card) error {
	resp := response{
		ID:          card.ID,
		Title:       card.Title,
		Description: card.Description,
	}

	return c.JSON(http.StatusOK, resp)
}

func writeError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, errInvalidCardID):
		slog.Error("invalid card id", "error", err)
		return c.String(http.StatusBadRequest, errInvalidCardID.Error()+"\n")
	case errors.Is(err, domain.ErrCardNotFound):
		return c.String(http.StatusNotFound, domain.ErrCardNotFound.Error()+"\n")
	default:
		slog.Error("failed to get card", "error", err)
		return c.String(http.StatusInternalServerError, "failed to get card\n")
	}
}
