package list

import (
	"log/slog"
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
	"github.com/labstack/echo/v5"
)

type cardResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

func writeResponse(c *echo.Context, cards []domain.Card) error {
	// An empty collection must encode as [] rather than null.
	resp := make([]cardResponse, 0, len(cards))
	for _, card := range cards {
		resp = append(resp, cardResponse{
			ID:          card.ID,
			Title:       card.Title,
			Description: card.Description,
		})
	}

	return c.JSON(http.StatusOK, resp)
}

func writeError(c *echo.Context, err error) error {
	slog.Error("failed to get cards", "error", err)
	return c.String(http.StatusInternalServerError, "failed to get cards\n")
}
