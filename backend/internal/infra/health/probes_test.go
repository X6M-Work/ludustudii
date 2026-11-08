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

	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
)

type pingFunc func(context.Context) error

func (f pingFunc) PingContext(ctx context.Context) error { return f(ctx) }

func TestProbes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		path     string
		ping     string
		stopping bool
		status   int
		body     string
		calls    int
	}{
		{"liveness ignores database failure", LivenessPath, "fail", false, 200, `{"status":"ok"}`, 0},
		{"liveness while stopping", LivenessPath, "fail", true, 200, `{"status":"ok"}`, 0},
		{"available", ReadinessPath, "ok", false, 200, `{"status":"ok"}`, 1},
		{"ping failed", ReadinessPath, "fail", false, 503, `{"status":"unavailable"}`, 1},
		{"ping timed out", ReadinessPath, "timeout", false, 503, `{"status":"unavailable"}`, 1},
		{"stopping", ReadinessPath, "ok", true, 503, `{"status":"unavailable"}`, 0},
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
				if tc.stopping {
					state.MarkStopping(t.Context())
				}
				mux := http.NewServeMux()
				NewProbes(state, func(_ http.ResponseWriter, _ *http.Request, err error) {
					t.Errorf("unexpected probe error: %v", err)
				}).Mount(mux)
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

func TestProbes_UnsupportedMethods(t *testing.T) {
	t.Parallel()
	for _, path := range []string{LivenessPath, ReadinessPath} {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions} {
			t.Run(method+" "+path, func(t *testing.T) {
				t.Parallel()
				state := NewState(slog.New(slog.DiscardHandler), pingFunc(func(context.Context) error {
					t.Error("unsupported method pinged the database")
					return nil
				}))
				mux := http.NewServeMux()
				calls := 0
				NewProbes(state, func(w http.ResponseWriter, _ *http.Request, err error) {
					calls++
					if !errors.Is(err, httpkit.ErrMethodNotAllowed) {
						t.Errorf("error: %v, want ErrMethodNotAllowed", err)
					}
					w.WriteHeader(http.StatusMethodNotAllowed)
				}).Mount(mux)
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, nil))
				if rec.Code != http.StatusMethodNotAllowed || calls != 1 {
					t.Fatalf("status/error handler calls: %d/%d, want 405/1", rec.Code, calls)
				}
				if rec.Header().Get("Allow") != "GET, HEAD" || rec.Header().Get("Cache-Control") != "no-store" {
					t.Fatalf("headers: %v", rec.Header())
				}
			})
		}
	}
}
