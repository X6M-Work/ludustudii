package app

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/infra/apidocs"
	"github.com/apotourlyan/ludustudii/internal/infra/logging"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
	"github.com/apotourlyan/ludustudii/internal/services/lcards"
)

// MaxBodyBytes is the largest request body the router accepts; larger bodies
// are rejected with 413 Payload Too Large.
const MaxBodyBytes = 1 << 20

// NewRouter builds the production handler with recovery, request IDs, request
// logging, body limits, API routes, optional reference routes and the SPA.
// Each request's logging context includes its request_id attribute.
func NewRouter(log *slog.Logger, clientApp *spa.ClientApp, apiDocs *apidocs.Handler, db *sql.DB) http.Handler {
	return newRouter(log, clientApp, apiDocs, db, nil)
}

// newRouter is NewRouter with an optional extra registration step, which
// tests use to add throwaway routes.
func newRouter(log *slog.Logger, clientApp *spa.ClientApp, apiDocs *apidocs.Handler, db *sql.DB, extra func(*httpkit.Router)) http.Handler {
	onError := func(w http.ResponseWriter, r *http.Request, err error) { httpErrorHandler(log, w, r, err) }
	rt := httpkit.NewRouter(onError)

	lcards.RegisterRoutes(rt, db)
	// Unknown API paths get the JSON 404 rather than the SPA. Both patterns are
	// needed, or the mux redirects /api to /api/.
	notFound := func(http.ResponseWriter, *http.Request) error { return httpkit.ErrNotFound }
	rt.HandleFunc("/api", notFound)
	rt.HandleFunc("/api/", notFound)
	if apiDocs != nil {
		apiDocs.RegisterRoutes(rt)
	}
	spa.NewHandler(clientApp).RegisterRoutes(rt)
	if extra != nil {
		extra(rt)
	}

	handler := httpkit.Chain(
		httpkit.Recover(onError),
		httpkit.RequestID(func(ctx context.Context, id string) context.Context {
			return logging.WithAttrs(ctx, slog.String("request_id", id))
		}),
		// RequestLogger reads the pattern the mux sets, so no middleware after it
		// may replace the request.
		httpkit.RequestLogger(log),
		// Handle panics before the logger records the final response.
		httpkit.Recover(onError),
		httpkit.BodyLimit(MaxBodyBytes, onError),
	)(rt)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(httpkit.NewResponseWriter(w, r), r)
	})
}
