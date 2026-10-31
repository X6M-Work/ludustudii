package app

import (
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
	"github.com/labstack/echo/v5"
)

// httpErrorHandler writes an erresp.ErrorResponse for errors returned by
// handlers and middleware. It keeps the error's status code, mapping 404, 405,
// 413 and 415 to their named codes and any other 4xx to "request_error". Errors
// without a status, and 5xx errors, are logged and reported as a 500
// "unexpected_error" so internal details never reach the client. HEAD
// requests get the status only. Nothing is written if the response has
// already been committed.
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
	case http.StatusUnsupportedMediaType:
		resp = erresp.UnsupportedMediaType()
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
