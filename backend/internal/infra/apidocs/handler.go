package apidocs

import (
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
	"github.com/apotourlyan/ludustudii/internal/kit/staticfile"
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

// RegisterRoutes registers the reference routes, including HEAD and plain-text
// method errors. The trailing slash redirects only at the exact /docs/ path.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc(Path, serveFile(h.docs.Page))
	mux.HandleFunc(Path+"/{$}", redirectToPath)
	mux.HandleFunc(Path+"/openapi.json", serveFile(h.docs.Spec))
	mux.HandleFunc(Path+"/scalar.js", serveFile(h.docs.Script))
}

func methodNotAllowed(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return false
	}
	w.Header().Set("Allow", "GET, HEAD")
	_ = httpkit.Text(w, http.StatusMethodNotAllowed, "method not allowed\n")
	return true
}

// redirectToPath redirects Path+"/" to Path, keeping the query. The redirect
// is temporary so browsers do not keep it once the reference is disabled.
func redirectToPath(w http.ResponseWriter, r *http.Request) {
	if methodNotAllowed(w, r) {
		return
	}
	target := Path
	if q := r.URL.RawQuery; q != "" {
		target += "?" + q
	}
	w.Header().Set("Location", target)
	w.WriteHeader(http.StatusFound)
}

func serveFile(f staticfile.File) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if methodNotAllowed(w, r) {
			return
		}
		f.Serve(w, r)
	}
}
