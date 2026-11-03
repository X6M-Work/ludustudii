package httpkit

import (
	"net/http"
	"slices"
	"strings"
)

// resourceMethods lists the methods Resource accepts, in Allow header order.
var resourceMethods = []string{
	http.MethodGet, http.MethodHead, http.MethodPost,
	http.MethodPut, http.MethodPatch, http.MethodDelete,
}

// Router registers routes on one http.ServeMux and reports handler errors
// through a shared ErrorHandler. It serves requests through the mux, so the
// matched pattern and path values are set on each request.
type Router struct {
	mux     *http.ServeMux
	onError ErrorHandler
}

// NewRouter returns a Router with an empty mux that reports errors to onError.
func NewRouter(onError ErrorHandler) *Router {
	return &Router{mux: http.NewServeMux(), onError: onError}
}

// ServeHTTP dispatches the request through the mux.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rt.mux.ServeHTTP(w, r)
}

// Handle registers h for an http.ServeMux pattern. Services register their
// endpoints with Resource instead.
func (rt *Router) Handle(pattern string, h http.Handler) {
	rt.mux.Handle(pattern, h)
}

// HandleFunc registers h for an http.ServeMux pattern, routing its errors to
// the Router's ErrorHandler.
func (rt *Router) HandleFunc(pattern string, h HandlerFunc) {
	rt.mux.Handle(pattern, Adapt(h, rt.onError))
}

// Resource registers each handler under "METHOD path" and a method-less
// pattern on path that returns the JSON 405 with an Allow header. A GET
// handler also serves HEAD. Methods other than GET, HEAD, POST, PUT, PATCH and
// DELETE panic, as does an empty map.
func (rt *Router) Resource(path string, handlers map[string]HandlerFunc) {
	if len(handlers) == 0 {
		panic("httpkit: Resource " + path + " has no handlers")
	}
	for method, h := range handlers {
		if !slices.Contains(resourceMethods, method) {
			panic("httpkit: Resource " + path + " has unsupported method " + method)
		}
		rt.HandleFunc(method+" "+path, h)
	}
	allowed := make([]string, 0, len(resourceMethods))
	for _, method := range resourceMethods {
		_, ok := handlers[method]
		_, get := handlers[http.MethodGet]
		if ok || method == http.MethodHead && get {
			allowed = append(allowed, method)
		}
	}
	allow := strings.Join(allowed, ", ")
	rt.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) error {
		w.Header().Set("Allow", allow)
		return ErrMethodNotAllowed
	})
}
