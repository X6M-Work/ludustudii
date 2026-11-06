//go:build integration

package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/infra/health"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
	"github.com/apotourlyan/ludustudii/internal/testkit"
)

func TestRouter_HealthEndpoints(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"database unavailable", "stopping with healthy database"} {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			db := testkit.NewDB(t)
			clientApp, err := spa.Load()
			if err != nil {
				t.Fatal(err)
			}
			log := discardLogger()
			state := health.NewState(log, db)
			srv := httptest.NewServer(NewRouter(log, clientApp, nil, db, state))
			t.Cleanup(srv.Close)
			probe := func(path string, status int, body string) {
				t.Helper()
				for _, method := range []string{http.MethodGet, http.MethodHead} {
					resp := testkit.Do(t, srv, testkit.DoOpts{Method: method, Path: path})
					testkit.AssertStatus(t, resp, status)
					if method == http.MethodGet {
						testkit.AssertJSON(t, resp.Body, body)
					} else if len(resp.Body) != 0 {
						t.Fatal("HEAD has a response body")
					}
					if resp.Header.Get("Cache-Control") != "no-store" {
						t.Fatal("missing Cache-Control: no-store")
					}
				}
			}

			probe(health.LivenessPath, 200, `{"status":"ok"}`)
			probe(health.ReadinessPath, 200, `{"status":"ok"}`)
			for _, path := range []string{health.LivenessPath, health.ReadinessPath} {
				for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions} {
					resp := testkit.Do(t, srv, testkit.DoOpts{Method: method, Path: path})
					testkit.AssertStatus(t, resp, http.StatusMethodNotAllowed)
					testkit.AssertJSON(t, resp.Body, `{"code":"method_not_allowed"}`)
					if resp.Header.Get("Allow") != "GET, HEAD" || resp.Header.Get("Cache-Control") != "no-store" || resp.Header.Get("X-Request-Id") == "" {
						t.Fatalf("method error headers: %v", resp.Header)
					}
				}
			}
			if scenario == "database unavailable" {
				if err := db.Close(); err != nil {
					t.Fatal(err)
				}
			} else {
				state.MarkStopping(t.Context())
				if err := db.PingContext(t.Context()); err != nil {
					t.Fatalf("database must stay healthy while stopping: %v", err)
				}
			}
			probe(health.ReadinessPath, 503, `{"status":"unavailable"}`)
			probe(health.LivenessPath, 200, `{"status":"ok"}`)
		})
	}
}
