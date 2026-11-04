package httpkit

import "net/http"

// API serves the API routes from its own http.ServeMux and reports handler
// errors through a shared ErrorHandler. The mux still decides 404 and 405, so
// method patterns like "GET /items/due" and "GET /items/{id}" coexist and the
// Allow header lists the methods the mux knows for the path; API only replaces
// the mux's plain-text bodies with ErrNotFound and ErrMethodNotAllowed.
type API struct {
	mux     *http.ServeMux
	onError ErrorHandler
}

// NewAPI returns an API with an empty mux that reports errors to onError.
func NewAPI(onError ErrorHandler) *API {
	return &API{mux: http.NewServeMux(), onError: onError}
}

// HandleFunc registers h for an http.ServeMux pattern, routing its errors to
// the API's ErrorHandler. A GET pattern also serves HEAD.
func (a *API) HandleFunc(pattern string, h HandlerFunc) {
	a.mux.Handle(pattern, Adapt(h, a.onError))
}

// ServeHTTP dispatches the request through the mux, which sets the matched
// pattern and path values. When no pattern matches, it runs the mux's own
// 404 or 405 handler against a probe to read the status and Allow header,
// then reports the matching error instead of the mux's text body.
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h, pattern := a.mux.Handler(r); pattern == "" {
		p := &probe{header: http.Header{}}
		h.ServeHTTP(p, r)
		if p.status == http.StatusMethodNotAllowed {
			w.Header().Set("Allow", p.header.Get("Allow"))
			a.onError(w, r, ErrMethodNotAllowed)
			return
		}
		a.onError(w, r, ErrNotFound)
		return
	}
	a.mux.ServeHTTP(w, r)
}

// probe records the status and headers a handler writes and discards its body.
type probe struct {
	header http.Header
	status int
}

func (p *probe) Header() http.Header { return p.header }

func (p *probe) Write(b []byte) (int, error) {
	if p.status == 0 {
		p.status = http.StatusOK
	}
	return len(b), nil
}

func (p *probe) WriteHeader(status int) {
	if p.status == 0 {
		p.status = status
	}
}
