//go:build unit

package health

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type pingFunc func(context.Context) error

func (f pingFunc) PingContext(ctx context.Context) error { return f(ctx) }

func TestProbes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		path     string
		ping     string
		draining bool
		status   int
		body     string
		calls    int
	}{
		{"liveness ignores database failure", LivenessPath, "fail", false, 200, `{"status":"ok"}`, 0},
		{"liveness during drain", LivenessPath, "fail", true, 200, `{"status":"ok"}`, 0},
		{"available", ReadinessPath, "ok", false, 200, `{"status":"ok"}`, 1},
		{"ping failed", ReadinessPath, "fail", false, 503, `{"status":"unavailable"}`, 1},
		{"ping timed out", ReadinessPath, "timeout", false, 503, `{"status":"unavailable"}`, 1},
		{"draining", ReadinessPath, "ok", true, 503, `{"status":"unavailable"}`, 0},
	} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			t.Run(tc.name+"/"+method, func(t *testing.T) {
				t.Parallel()
				calls := 0
				state := NewState(slog.New(slog.DiscardHandler), pingFunc(func(ctx context.Context) error {
					calls++
					deadline, ok := ctx.Deadline()
					if !ok || time.Until(deadline) > time.Second {
						t.Error("ping must have a deadline within one second")
					}
					switch tc.ping {
					case "fail":
						return errors.New("private database error")
					case "timeout":
						<-ctx.Done()
						return ctx.Err()
					default:
						return nil
					}
				}))
				if tc.draining {
					state.StartDraining(t.Context())
				}
				mux := http.NewServeMux()
				RegisterProbes(mux, state)
				rec := httptest.NewRecorder()
				start := time.Now()
				mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, tc.path, nil))
				if tc.ping == "timeout" && time.Since(start) > 2*time.Second {
					t.Error("ping timeout took too long")
				}
				wantBody := tc.body
				if method == http.MethodHead {
					wantBody = ""
				}
				if rec.Code != tc.status || rec.Body.String() != wantBody {
					t.Fatalf("response: %d %q, want %d %q", rec.Code, rec.Body.String(), tc.status, wantBody)
				}
				if calls != tc.calls {
					t.Errorf("ping calls: %d, want %d", calls, tc.calls)
				}
				if rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("Content-Type") != "application/json" {
					t.Fatalf("headers: %v", rec.Header())
				}
			})
		}
	}
}
