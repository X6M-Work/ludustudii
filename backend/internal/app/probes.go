package app

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	livenessPath         = "/livez"
	readinessPath        = "/readyz"
	readinessPingTimeout = time.Second
)

type databasePinger interface {
	PingContext(context.Context) error
}

// readiness shares the irreversible drain switch with shutdown and records
// database availability transitions. The mutex serializes state changes and
// their logs, but is never held during a database ping.
type readiness struct {
	db          databasePinger
	log         *slog.Logger
	draining    atomic.Bool
	mu          sync.Mutex
	unavailable bool
}

func newReadiness(log *slog.Logger, db databasePinger) *readiness {
	return &readiness{db: db, log: log}
}

func (s *readiness) markNotReady(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.draining.Swap(true) {
		s.log.InfoContext(ctx, "Readiness draining started")
	}
}

func (s *readiness) ready(ctx context.Context) bool {
	if s.draining.Load() {
		return false
	}
	pingCtx, cancel := context.WithTimeout(ctx, readinessPingTimeout)
	defer cancel()
	err := s.db.PingContext(pingCtx)
	// A ping that finishes after its deadline must never report readiness.
	if err == nil {
		err = pingCtx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// An in-flight ping must not restore readiness after draining starts.
	if s.draining.Load() {
		return false
	}
	unavailable := err != nil
	if unavailable != s.unavailable {
		s.unavailable = unavailable
		if unavailable {
			s.log.WarnContext(ctx, "Readiness unavailable", "error", err)
		} else {
			s.log.InfoContext(ctx, "Readiness recovered")
		}
	}
	return !unavailable
}

func (s *readiness) registerProbes(mux *http.ServeMux) {
	probes := http.NewServeMux()
	probes.HandleFunc("GET "+livenessPath, func(w http.ResponseWriter, r *http.Request) {
		writeProbe(w, r, true)
	})
	probes.HandleFunc("GET "+readinessPath, func(w http.ResponseWriter, r *http.Request) {
		writeProbe(w, r, s.ready(r.Context()))
	})
	// Reserve these exact paths for all methods so the SPA cannot catch them.
	// The inner mux supplies the standard non-API 405 and Allow header.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		probes.ServeHTTP(w, r)
	})
	mux.Handle(livenessPath, handler)
	mux.Handle(readinessPath, handler)
}

func writeProbe(w http.ResponseWriter, r *http.Request, ready bool) {
	w.Header().Set("Content-Type", "application/json")
	status, body := http.StatusOK, `{"status":"ok"}`
	if !ready {
		status, body = http.StatusServiceUnavailable, `{"status":"unavailable"}`
	}
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		// Fixed JSON cannot fail to encode; a disconnected client needs no
		// additional log record and cannot receive a second response.
		_, _ = w.Write([]byte(body))
	}
}
