package app

import (
	"log/slog"
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
)

// httpErrorHandler writes an erresp.ErrorResponse for errors returned by
// handlers and middleware. It keeps the error's status code, mapping 404, 405,
// 413 and 415 to their named codes and any other 4xx to "request_error". Errors
// without a status, and 5xx errors, are logged and reported as a 500
// "unexpected_error" so internal details never reach the client. HEAD
// requests get the status only. Nothing is written if the response has
// already been committed.
func httpErrorHandler(log *slog.Logger, w http.ResponseWriter, r *http.Request, err error) {
	if httpkit.IsCommitted(w) {
		return
	}

	code := httpkit.StatusCode(err)
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
			log.ErrorContext(r.Context(), "Unhandled error",
				"error", err, "method", r.Method, "path", r.URL.Path)
			code = http.StatusInternalServerError
			resp = erresp.Unexpected()
		} else {
			resp = erresp.New("request_error") // keeps the status; add a named case when one matters
		}
	}

	var cErr error
	if r.Method == http.MethodHead {
		cErr = httpkit.NoContent(w, code)
	} else {
		cErr = httpkit.JSON(w, code, resp)
	}
	if cErr != nil {
		log.ErrorContext(r.Context(), "Failed to send error response", "error", cErr)
	}
}
