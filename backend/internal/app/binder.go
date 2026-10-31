package app

import (
	"strings"

	"github.com/labstack/echo/v5"
)

// jsonBinder is the router-wide echo.Binder. It binds only JSON request
// bodies and ignores path and query parameters, which handlers bind
// explicitly when they need them.
type jsonBinder struct{}

// Bind decodes the request body into target. A request with a body (or of
// unknown length) must have a Content-Type of application/json, parameters
// aside, or Bind returns echo.ErrUnsupportedMediaType. An empty body leaves
// target unchanged.
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
