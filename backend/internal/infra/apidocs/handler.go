package apidocs

import (
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
)

// Path is where the reference page is served. The specification and the
// Scalar script are served below it, and the page loads them by URLs relative
// to Path, so dist/index.html names them after Path's last segment.
const Path = "/docs"

// Handler serves the reference page, the bundled specification and the Scalar
// script for GET and HEAD requests. None of them is content-hashed in its URL,
// so all are revalidated with their ETags.
type Handler struct {
	docs *Docs
}

// NewHandler returns a Handler that serves docs.
func NewHandler(docs *Docs) *Handler {
	return &Handler{docs: docs}
}

// RegisterRoutes registers the reference routes for GET and HEAD. Other
// methods fall through to the router's catch-all route. The trailing slash
// redirects only at the exact /docs/ path.
func (h *Handler) RegisterRoutes(rt *httpkit.Router) {
	rt.Handle("GET "+Path, http.HandlerFunc(h.docs.Page.Serve))
	rt.Handle("GET "+Path+"/{$}", http.HandlerFunc(redirectToPath))
	rt.Handle("GET "+Path+"/openapi.json", http.HandlerFunc(h.docs.Spec.Serve))
	rt.Handle("GET "+Path+"/scalar.js", http.HandlerFunc(h.docs.Script.Serve))
}

// redirectToPath redirects Path+"/" to Path, keeping the query. The redirect
// is temporary so browsers do not keep it once the reference is disabled.
func redirectToPath(w http.ResponseWriter, r *http.Request) {
	target := Path
	if q := r.URL.RawQuery; q != "" {
		target += "?" + q
	}
	w.Header().Set("Location", target)
	w.WriteHeader(http.StatusFound)
}
