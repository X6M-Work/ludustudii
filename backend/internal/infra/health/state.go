// Package health provides liveness and readiness probes with shared availability state.
package health

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const readinessPingTimeout = time.Second

type databasePinger interface {
	PingContext(context.Context) error
}

type state uint8

const (
	available state = iota
	unavailable
	draining
)

// State tracks availability: available, unavailable, or draining. Database pings
// can move between available and unavailable; draining is permanent. The mutex
// protects state changes and their logs; database pings run without holding it.
type State struct {
	db    databasePinger
	log   *slog.Logger
	mu    sync.Mutex
	state state
}

// NewState returns an available State for a database connected during startup.
func NewState(log *slog.Logger, db databasePinger) *State {
	return &State{db: db, log: log, state: available}
}

// StartDraining permanently marks the instance unavailable before HTTP shutdown.
func (s *State) StartDraining(ctx context.Context) {
	s.transition(ctx, draining, nil)
}

// Check pings the database within one second and reports whether the instance
// can receive traffic. Once draining starts, Check returns false without pinging.
func (s *State) Check(ctx context.Context) bool {
	if s.currentState() == draining {
		return false
	}

	pingCtx, cancel := context.WithTimeout(ctx, readinessPingTimeout)
	defer cancel()
	err := s.db.PingContext(pingCtx)
	if err == nil {
		err = pingCtx.Err()
	}

	next := available
	if err != nil {
		next = unavailable
	}
	return s.transition(ctx, next, err) == available
}

func (s *State) currentState() state {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// transition is the only place that changes health state and logs state changes.
// Repeated states are silent. Draining rejects all subsequent transitions,
// including results of pings that were in flight when shutdown started.
func (s *State) transition(ctx context.Context, next state, reason error) state {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == draining || s.state == next {
		return s.state
	}

	s.state = next
	switch next {
	case available:
		s.log.InfoContext(ctx, "Readiness recovered")
	case unavailable:
		s.log.WarnContext(ctx, "Readiness unavailable", "error", reason)
	case draining:
		s.log.InfoContext(ctx, "Readiness draining started")
	}
	return s.state
}
