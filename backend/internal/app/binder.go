package app

import (
	"strings"

	"github.com/labstack/echo/v5"
)

type jsonBinder struct{}

func (jsonBinder) Bind(c *echo.Context, target any) error {
	req := c.Request()
	if req.ContentLength != 0 {
		base, _, _ := strings.Cut(req.Header.Get(echo.HeaderContentType), ";")
		if strings.TrimSpace(base) != echo.MIMEApplicationJSON {
			return echo.ErrUnsupportedMediaType
		}
	}
	return echo.BindBody(c, target)
}
