// Package router is the shared HTTP front of a server binary: one root mux,
// the middleware chain every request passes through and the central error
// handler. It knows no surface. A composition root builds the surfaces it
// serves and mounts them with New, so the same front works for any subset of
// them.
package router

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/infra/logging"
	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
)

// MaxBodyBytes is the largest request body the router accepts; larger bodies
// are rejected with 413 Payload Too Large on every surface.
const MaxBodyBytes = 1 << 20

// Surface is an independently built part of the HTTP server, such as the
// probes, the API or the client app. It owns a set of patterns on the root
// mux and registers them in Mount. Surfaces must not register conflicting
// patterns; http.ServeMux panics when they do.
type Surface interface {
	Mount(mux *http.ServeMux)
}

// QuietSurface is a Surface polled often enough, such as by a container
// platform, that its request summaries log at debug level at every status.
type QuietSurface interface {
	Surface
	// QuietPaths returns the exact URL paths whose summaries are quiet.
	QuietPaths() []string
}

// New mounts surfaces on one root mux and wraps it with recovery, request IDs,
// request logging and the body limit, reporting errors to onError. Each
// request's logging context includes its request_id attribute. Requests that
// match no surface get the mux's own 404 or 405.
func New(log *slog.Logger, onError httpkit.ErrorHandler, surfaces ...Surface) http.Handler {
	mux := http.NewServeMux()
	quiet := map[string]struct{}{}
	for _, s := range surfaces {
		s.Mount(mux)
		if q, ok := s.(QuietSurface); ok {
			for _, path := range q.QuietPaths() {
				quiet[path] = struct{}{}
			}
		}
	}
	return withMiddleware(log, onError, quiet, mux)
}

// ErrorHandler returns the central error handler, which writes the shared JSON
// error shape and logs with log. Surfaces report their errors to it.
func ErrorHandler(log *slog.Logger) httpkit.ErrorHandler {
	return func(w http.ResponseWriter, r *http.Request, err error) { httpErrorHandler(log, w, r, err) }
}

// withMiddleware wraps h with the production middleware chain, reporting
// errors to onError. Requests to quietPaths log their summaries at debug.
func withMiddleware(log *slog.Logger, onError httpkit.ErrorHandler, quietPaths map[string]struct{}, h http.Handler) http.Handler {
	handler := httpkit.Chain(
		httpkit.Recover(onError),
		httpkit.RequestID(func(ctx context.Context, id string) context.Context {
			return logging.WithAttrs(ctx, slog.String("request_id", id))
		}),
		// RequestLogger reads the pattern the muxes set on the request, so no
		// middleware after it may replace the request.
		httpkit.RequestLogger(log, httpkit.RequestLoggerOptions{QuietPaths: quietPaths}),
		// Handle panics before the logger records the final response.
		httpkit.Recover(onError),
		httpkit.BodyLimit(MaxBodyBytes, onError),
	)(h)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(httpkit.NewResponseWriter(w, r), r)
	})
}
