package app

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/infra/apidocs"
	"github.com/apotourlyan/ludustudii/internal/infra/health"
	"github.com/apotourlyan/ludustudii/internal/infra/logging"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
	"github.com/apotourlyan/ludustudii/internal/services/lcards"
)

// MaxBodyBytes is the largest request body the router accepts; larger bodies
// are rejected with 413 Payload Too Large.
const MaxBodyBytes = 1 << 20

// NewRouter builds the production handler with recovery, request IDs, request
// logging, probes, body limits, API routes, optional reference routes and the SPA.
// Each request's logging context includes its request_id attribute.
func NewRouter(log *slog.Logger, clientApp *spa.ClientApp, apiDocs *apidocs.Handler, db *sql.DB, state *health.State) http.Handler {
	return newRouter(log, clientApp, apiDocs, db, state)
}

func newRouter(log *slog.Logger, clientApp *spa.ClientApp, apiDocs *apidocs.Handler, db *sql.DB, state *health.State) http.Handler {
	onError := errorHandler(log)

	api := httpkit.NewAPI(onError)
	lcards.RegisterRoutes(api, db)

	mux := http.NewServeMux()
	health.RegisterProbes(mux, state, onError)
	// Every /api path goes to api, which answers unknown paths and methods
	// with JSON rather than the SPA. Both patterns are needed, or the mux
	// redirects /api to /api/.
	mux.Handle("/api", api)
	mux.Handle("/api/", api)
	if apiDocs != nil {
		apiDocs.RegisterRoutes(mux)
	}
	spa.NewHandler(clientApp).RegisterRoutes(mux, onError)

	return withMiddleware(log, onError, mux)
}

// errorHandler reports request errors through httpErrorHandler with log.
func errorHandler(log *slog.Logger) httpkit.ErrorHandler {
	return func(w http.ResponseWriter, r *http.Request, err error) { httpErrorHandler(log, w, r, err) }
}

// withMiddleware wraps h with the production middleware chain, reporting
// errors to onError.
func withMiddleware(log *slog.Logger, onError httpkit.ErrorHandler, h http.Handler) http.Handler {
	handler := httpkit.Chain(
		httpkit.Recover(onError),
		httpkit.RequestID(func(ctx context.Context, id string) context.Context {
			return logging.WithAttrs(ctx, slog.String("request_id", id))
		}),
		// RequestLogger reads the pattern the muxes set on the request, so no
		// middleware after it may replace the request.
		httpkit.RequestLogger(log, httpkit.RequestLoggerOptions{
			QuietPaths: map[string]struct{}{health.LivenessPath: {}, health.ReadinessPath: {}},
		}),
		// Handle panics before the logger records the final response.
		httpkit.Recover(onError),
		httpkit.BodyLimit(MaxBodyBytes, onError, httpkit.BodyLimitOptions{
			ExcludedPaths: map[string]struct{}{health.LivenessPath: {}, health.ReadinessPath: {}},
		}),
	)(h)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(httpkit.NewResponseWriter(w, r), r)
	})
}
