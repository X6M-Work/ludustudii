package health

import "net/http"

const (
	// LivenessPath is the dependency-free process liveness endpoint.
	LivenessPath = "/livez"
	// ReadinessPath is the database-backed traffic readiness endpoint.
	ReadinessPath = "/readyz"
)

// RegisterProbes reserves the probe paths for GET and HEAD, returning 405 for
// other methods rather than letting them reach a catch-all handler.
func RegisterProbes(mux *http.ServeMux, state *State) {
	// Reserve these exact paths for all methods so the SPA cannot catch them.
	mux.HandleFunc(LivenessPath, probeHandler(func(w http.ResponseWriter, r *http.Request) {
		writeProbe(w, r, true)
	}))
	mux.HandleFunc(ReadinessPath, probeHandler(func(w http.ResponseWriter, r *http.Request) {
		writeProbe(w, r, state.Check(r.Context()))
	}))
}

// probeHandler applies the common HTTP headers and method restrictions.
func probeHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		next(w, r)
	}
}

func writeProbe(w http.ResponseWriter, r *http.Request, available bool) {
	w.Header().Set("Content-Type", "application/json")
	status, body := http.StatusOK, `{"status":"ok"}`
	if !available {
		status, body = http.StatusServiceUnavailable, `{"status":"unavailable"}`
	}
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		// Fixed JSON cannot fail to encode; a disconnected client needs no
		// additional log record and cannot receive a second response.
		_, _ = w.Write([]byte(body))
	}
}
