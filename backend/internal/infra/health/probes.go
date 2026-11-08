package health

import (
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
)

const (
	// LivenessPath is the dependency-free process liveness endpoint.
	LivenessPath = "/livez"
	// ReadinessPath is the database-backed traffic readiness endpoint.
	ReadinessPath = "/readyz"
)

// Probes is the operational surface: the liveness and readiness endpoints at
// the root, outside the API and the client app. It depends only on State, so
// every server binary can mount it.
type Probes struct {
	state   *State
	onError httpkit.ErrorHandler
}

// NewProbes returns Probes that report readiness from state and method errors
// through onError.
func NewProbes(state *State, onError httpkit.ErrorHandler) *Probes {
	return &Probes{state: state, onError: onError}
}

// Mount reserves the probe paths for GET and HEAD, returning 405 for other
// methods through onError rather than reaching a catch-all route.
func (p *Probes) Mount(mux *http.ServeMux) {
	// Reserve these exact paths for all methods so a catch-all cannot catch them.
	mux.HandleFunc(LivenessPath, probeHandler(func(w http.ResponseWriter, r *http.Request) {
		writeProbe(w, r, true)
	}, p.onError))
	mux.HandleFunc(ReadinessPath, probeHandler(func(w http.ResponseWriter, r *http.Request) {
		writeProbe(w, r, p.state.Check(r.Context()))
	}, p.onError))
}

// QuietPaths returns the probe paths, whose request summaries log at debug.
func (p *Probes) QuietPaths() []string {
	return []string{LivenessPath, ReadinessPath}
}

// probeHandler applies the common HTTP headers and method restrictions.
func probeHandler(next http.HandlerFunc, onError httpkit.ErrorHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			onError(w, r, httpkit.ErrMethodNotAllowed)
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
