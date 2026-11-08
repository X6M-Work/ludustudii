package router

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/kit"
	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
)

// httpErrorHandler sends status errors as supplied and logs unexpected errors
// before returning a generic 500. HEAD responses omit the body and committed
// responses are left alone.
func httpErrorHandler(log *slog.Logger, w http.ResponseWriter, r *http.Request, err error) {
	if httpkit.IsCommitted(w) {
		return
	}

	se, ok := toStatusError(err)
	if !ok {
		log.ErrorContext(r.Context(), "Unhandled error", "error", err, "method", r.Method, "path", r.URL.Path)
		se = httpkit.ErrUnexpected
	} else {
		log.DebugContext(r.Context(), "Request failed", "error", err, "method", r.Method, "path", r.URL.Path)
	}

	if r.Method == http.MethodHead {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(se.Code)
		return
	}
	if err := httpkit.JSON(w, se.Code, se.Response); err != nil {
		log.ErrorContext(r.Context(), "Failed to send error response", "error", err)
	}
}

// toStatusError maps storage sentinels to HTTP; anything else must already be a StatusError.
func toStatusError(err error) (*httpkit.StatusError, bool) {
	switch {
	case errors.Is(err, kit.ErrNotFound):
		return httpkit.ErrNotFound, true
	case errors.Is(err, kit.ErrVersionConflict):
		return httpkit.ErrPreconditionFailed, true
	}
	return errors.AsType[*httpkit.StatusError](err)
}
