package app

import (
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
	"github.com/labstack/echo/v5"
)

func httpErrorHandler(c *echo.Context, err error) {
	if r, _ := echo.UnwrapResponse(c.Response()); r != nil && r.Committed {
		return
	}

	code := echo.StatusCode(err)
	var resp erresp.ErrorResponse
	switch code {
	case http.StatusNotFound:
		resp = erresp.NotFound()
	case http.StatusMethodNotAllowed:
		resp = erresp.MethodNotAllowed()
	case http.StatusRequestEntityTooLarge:
		resp = erresp.PayloadTooLarge()
	default:
		if code == 0 || code >= http.StatusInternalServerError {
			c.Logger().ErrorContext(c.Request().Context(), "Unhandled error",
				"error", err, "method", c.Request().Method, "path", c.Request().URL.Path)
			code = http.StatusInternalServerError
			resp = erresp.Unexpected()
		} else {
			resp = erresp.New("request_error") // keeps the status; add a named case when one matters
		}
	}

	var cErr error
	if c.Request().Method == http.MethodHead {
		cErr = c.NoContent(code)
	} else {
		cErr = c.JSON(code, resp)
	}
	if cErr != nil {
		c.Logger().Error("Failed to send error response", "error", cErr)
	}
}
