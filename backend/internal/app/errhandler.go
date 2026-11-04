package app

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/kit"
	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
)

// httpErrorHandler maps returned request and storage errors to the shared JSON
// shape. Client errors are logged at debug level, unexpected errors at error
// level. HEAD responses omit the body and committed responses are left alone.
func httpErrorHandler(log *slog.Logger, w http.ResponseWriter, r *http.Request, err error) {
	if httpkit.IsCommitted(w) {
		return
	}

	code := httpkit.StatusCode(err)
	switch {
	case errors.Is(err, kit.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, kit.ErrVersionConflict):
		code = http.StatusPreconditionFailed
	}
	var resp erresp.ErrorResponse
	switch code {
	case http.StatusPreconditionFailed:
		resp = erresp.PreconditionFailed()
	case http.StatusNotFound:
		resp = erresp.NotFound()
	case http.StatusMethodNotAllowed:
		resp = erresp.MethodNotAllowed()
	case http.StatusRequestEntityTooLarge:
		resp = erresp.PayloadTooLarge()
	case http.StatusUnsupportedMediaType:
		resp = erresp.UnsupportedMediaType()
	default:
		if code < http.StatusBadRequest || code >= http.StatusInternalServerError {
			log.ErrorContext(r.Context(), "Unhandled error",
				"error", err, "method", r.Method, "path", r.URL.Path)
			code = http.StatusInternalServerError
			resp = erresp.Unexpected()
		} else {
			resp = erresp.New("request_error")
			if typed, ok := errors.AsType[*httpkit.StatusError](err); ok && typed.Response.Code != "" {
				resp = typed.Response
			}
		}
	}

	if code < http.StatusInternalServerError {
		cause := err
		if typed, ok := errors.AsType[*httpkit.StatusError](err); ok && typed.Err != nil {
			cause = typed.Err
		}
		log.DebugContext(r.Context(), "Request failed", "error", cause, "method", r.Method, "path", r.URL.Path)
	}

	var cErr error
	if r.Method == http.MethodHead {
		w.Header().Set("Content-Type", "application/json")
		cErr = httpkit.NoContent(w, code)
	} else {
		cErr = httpkit.JSON(w, code, resp)
	}
	if cErr != nil {
		log.ErrorContext(r.Context(), "Failed to send error response", "error", cErr)
	}
}
