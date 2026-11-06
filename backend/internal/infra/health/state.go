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
	stopping
)

// State tracks availability: available, unavailable, or stopping. Database pings
// can move between available and unavailable; stopping is permanent. The mutex
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

// MarkStopping permanently marks the instance unavailable before HTTP shutdown.
func (s *State) MarkStopping(ctx context.Context) {
	s.transition(ctx, stopping, nil)
}

// Check pings the database within one second and reports whether the instance
// can receive traffic. Once stopping starts, Check returns false without pinging.
// A canceled request returns false without changing availability or logging.
func (s *State) Check(ctx context.Context) bool {
	if s.currentState() == stopping {
		return false
	}

	pingCtx, cancel := context.WithTimeout(ctx, readinessPingTimeout)
	defer cancel()
	err := s.db.PingContext(pingCtx)
	// An abandoned probe does not establish whether the database is available.
	if ctx.Err() != nil {
		return false
	}
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
// Repeated states are silent. Stopping rejects all subsequent transitions,
// including results of pings that were in flight when shutdown started.
func (s *State) transition(ctx context.Context, next state, reason error) state {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == stopping || s.state == next {
		return s.state
	}

	s.state = next
	switch next {
	case available:
		s.log.InfoContext(ctx, "Readiness recovered")
	case unavailable:
		s.log.WarnContext(ctx, "Readiness unavailable", "error", reason)
	case stopping:
		s.log.InfoContext(ctx, "Readiness stopping")
	}
	return s.state
}
