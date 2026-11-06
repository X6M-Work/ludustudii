//go:build integration

package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/infra/health"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
	"github.com/apotourlyan/ludustudii/internal/testkit"
)

func TestMain(m *testing.M) {
	os.Exit(testkit.Run(m))
}

func TestProbes_RealDatabase(t *testing.T) {
	t.Parallel()
	db := testkit.NewDB(t)
	clientApp, err := spa.Load()
	if err != nil {
		t.Fatal(err)
	}
	log := discardLogger()
	state := health.NewState(log, db)
	srv := httptest.NewServer(newRouter(log, clientApp, nil, db, state))
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
		resp := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodPost, Path: path})
		testkit.AssertStatus(t, resp, http.StatusMethodNotAllowed)
		if resp.Header.Get("Allow") != "GET, HEAD" {
			t.Fatalf("Allow: %q", resp.Header.Get("Allow"))
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	probe(health.ReadinessPath, 503, `{"status":"unavailable"}`)
	probe(health.LivenessPath, 200, `{"status":"ok"}`)
	state.StartDraining(t.Context())
	probe(health.ReadinessPath, 503, `{"status":"unavailable"}`)
	probe(health.LivenessPath, 200, `{"status":"ok"}`)
}
