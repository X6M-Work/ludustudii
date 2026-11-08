//go:build integration

package router

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
	"github.com/apotourlyan/ludustudii/internal/testkit"
)

func TestMain(m *testing.M) { os.Exit(testkit.Run(m)) }

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestMiddleware_PanicResponse(t *testing.T) {
	t.Parallel()
	log := discardLogger()
	onError := ErrorHandler(log)
	api := httpkit.NewAPI(onError)
	api.HandleFunc("GET /api/panic", func(http.ResponseWriter, *http.Request) error { panic("boom") })
	srv := httptest.NewServer(withMiddleware(log, onError, nil, api))
	t.Cleanup(srv.Close)
	resp := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: "/api/panic"})
	testkit.AssertStatus(t, resp, http.StatusInternalServerError)
	testkit.AssertJSON(t, resp.Body, `{"code":"unexpected_error"}`)
}

func TestMiddleware_ClientAppPanicResponse(t *testing.T) {
	t.Parallel()
	log := discardLogger()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(http.ResponseWriter, *http.Request) { panic("boom") })
	srv := httptest.NewServer(withMiddleware(log, ErrorHandler(log), nil, mux))
	t.Cleanup(srv.Close)
	resp := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: "/client"})
	testkit.AssertStatus(t, resp, http.StatusInternalServerError)
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", ct)
	}
	testkit.AssertJSON(t, resp.Body, `{"code":"unexpected_error"}`)
}
