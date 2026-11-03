package app

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

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
	onError := func(w http.ResponseWriter, r *http.Request, err error) { httpErrorHandler(log, w, r, err) }
	mux := http.NewServeMux()
	lcards.RegisterRoutes(mux, db, onError)
	if apiDocs != nil {
		apiDocs.RegisterRoutes(mux)
	}
	fallback := http.NewServeMux()
	spa.NewHandler(clientApp).RegisterRoutes(fallback, onError)
	rt := &router{mux: mux, fallback: fallback, onError: onError}

	rt.wrap = httpkit.Chain(
		httpkit.Recover(onError),
		httpkit.RequestID(func(ctx context.Context, id string) context.Context {
			return logging.WithAttrs(ctx, slog.String("request_id", id))
		}),
		httpkit.RequestLogger(log),
		// Handle panics before the logger records the final response.
		httpkit.Recover(onError),
		httpkit.BodyLimit(MaxBodyBytes, onError),
	)

	return rt
}

type router struct {
	mux      *http.ServeMux
	fallback *http.ServeMux
	onError  httpkit.ErrorHandler
	wrap     httpkit.Middleware
}

func (rt *router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler := rt.resolve(r)
	rt.wrap(handler).ServeHTTP(httpkit.NewResponseWriter(w, r), r)
}

func (rt *router) resolve(r *http.Request) http.Handler {
	// Match the original spelling of escaped paths. In particular, an escaped
	// static segment must not become an API route after unescaping.
	routePath := r.URL.Path
	if r.URL.RawPath != "" {
		routePath = r.URL.RawPath
	}
	lookup := *r
	lookupURL := *r.URL
	lookupURL.Path = routePath
	lookupURL.RawPath = ""
	// Protect literal dot segments from the mux's automatic cleaning redirects.
	segments := strings.Split(lookupURL.EscapedPath(), "/")
	for i, segment := range segments {
		switch segment {
		case ".":
			segments[i] = "%2e"
		case "..":
			segments[i] = "%2e%2e"
		}
	}
	lookupURL.RawPath = strings.Join(segments, "/")
	lookup.URL = &lookupURL
	if !strings.Contains(routePath, "//") {
		h, pattern := rt.mux.Handler(&lookup)
		if pattern != "" {
			r.Pattern = pattern
			// Handler resolves a route without populating wildcard values. The API's
			// single-segment id uses its original spelling, including percent escapes.
			if strings.HasSuffix(pattern, "/{id}") {
				r.SetPathValue("id", routePath[strings.LastIndexByte(routePath, '/')+1:])
			}
			return h
		}
		probe := &routeProbe{header: make(http.Header)}
		h.ServeHTTP(probe, &lookup)
		if probe.status == http.StatusMethodNotAllowed {
			// Resolve a GET solely to retain the route label for method errors.
			lookup.Method = http.MethodGet
			_, r.Pattern = rt.mux.Handler(&lookup)
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Allow", orderedAllow(probe.header.Get("Allow")))
				rt.onError(w, r, httpkit.ErrMethodNotAllowed)
			})
		}
	}
	if routePath == "/api" || strings.HasPrefix(routePath, "/api/") {
		r.Pattern = "/api/*"
		if routePath == "/api" {
			r.Pattern = "/api"
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rt.onError(w, r, httpkit.ErrNotFound)
		})
	}
	// Resolve the SPA registration at root, bypassing path-cleaning redirects.
	lookup.URL = &url.URL{Path: "/"}
	h, _ := rt.fallback.Handler(&lookup)
	r.Pattern = "/*"
	return h
}

// routeProbe captures the mux's synthetic response without sending its text body.
type routeProbe struct {
	header http.Header
	status int
}

func (p *routeProbe) Header() http.Header         { return p.header }
func (p *routeProbe) WriteHeader(status int)      { p.status = status }
func (p *routeProbe) Write(b []byte) (int, error) { return len(b), nil }

func orderedAllow(value string) string {
	methods := strings.Split(value, ", ")
	allowed := make(map[string]bool, len(methods))
	for _, method := range methods {
		allowed[method] = true
	}
	result := make([]string, 0, len(methods))
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodDelete} {
		if allowed[method] {
			result = append(result, method)
			delete(allowed, method)
		}
	}
	for _, method := range methods {
		if allowed[method] {
			result = append(result, method)
		}
	}
	return strings.Join(result, ", ")
}
