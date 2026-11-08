package apidocs

import "net/http"

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

// Mount registers the reference routes on mux for GET and HEAD, making Handler
// a surface of its own that needs neither the API nor the client app. Other
// methods get the mux's 405, or fall through to a catch-all route if another
// surface registers one. The trailing slash redirects only at the exact /docs/
// path.
func (h *Handler) Mount(mux *http.ServeMux) {
	mux.Handle("GET "+Path, http.HandlerFunc(h.docs.Page.Serve))
	mux.Handle("GET "+Path+"/{$}", http.HandlerFunc(redirectToPath))
	mux.Handle("GET "+Path+"/openapi.json", http.HandlerFunc(h.docs.Spec.Serve))
	mux.Handle("GET "+Path+"/scalar.js", http.HandlerFunc(h.docs.Script.Serve))
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
