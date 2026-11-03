package httpkit

import "net/http"

// ResponseWriter tracks the final status and whether headers have been committed.
// Unwrap preserves access to optional interfaces through ResponseController.
type ResponseWriter struct {
	http.ResponseWriter
	Status    int
	Committed bool
	head      bool
}

// NewResponseWriter wraps w and suppresses bodies for HEAD requests.
func NewResponseWriter(w http.ResponseWriter, r *http.Request) *ResponseWriter {
	return &ResponseWriter{ResponseWriter: w, Status: http.StatusOK, head: r.Method == http.MethodHead}
}

// Unwrap returns the underlying writer for http.ResponseController.
func (w *ResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// WriteHeader records the first final status. Informational headers do not commit.
func (w *ResponseWriter) WriteHeader(status int) {
	if w.Committed {
		return
	}
	if status >= 200 {
		w.Status = status
		w.Committed = true
	}
	w.ResponseWriter.WriteHeader(status)
}

// Write commits status 200 if needed, and omits body bytes for HEAD.
func (w *ResponseWriter) Write(p []byte) (int, error) {
	if !w.Committed {
		w.WriteHeader(http.StatusOK)
	}
	if w.head {
		return len(p), nil
	}
	return w.ResponseWriter.Write(p)
}

// Flush commits headers before flushing through the underlying controller.
func (w *ResponseWriter) Flush() { _ = w.FlushError() }

// FlushError commits headers and forwards a flush, preserving controller errors.
func (w *ResponseWriter) FlushError() error {
	if !w.Committed {
		w.WriteHeader(http.StatusOK)
	}
	return http.NewResponseController(w.ResponseWriter).Flush()
}

// IsCommitted reports whether w or one of its wrappers has committed headers.
func IsCommitted(w http.ResponseWriter) bool {
	for {
		if tracked, ok := w.(*ResponseWriter); ok {
			return tracked.Committed
		}
		unwrapped, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			return false
		}
		w = unwrapped.Unwrap()
	}
}
