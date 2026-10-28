package app

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/spa"
	"github.com/apotourlyan/ludustudii/internal/testkit"
	"github.com/labstack/echo/v5"
)

func newRouterServer(t *testing.T) *httptest.Server {
	t.Helper()
	clientApp, err := spa.Load()
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(NewRouter(clientApp, nil))
	t.Cleanup(srv.Close)
	return srv
}

func TestNewRouter(t *testing.T) {
	t.Parallel()

	t.Run("GET / serves SPA index.html", func(t *testing.T) {
		t.Parallel()
		srv := newRouterServer(t)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodGet,
			Path:   "/",
		})
		testkit.AssertStatus(t, resp, body, http.StatusOK)
		if ct := resp.Header.Get(echo.HeaderContentType); !strings.HasPrefix(ct, echo.MIMETextHTML) {
			t.Errorf("Content-Type: got %q, want %s", ct, echo.MIMETextHTML)
		}
		if len(body) == 0 {
			t.Error("expected non-empty HTML body")
		}
	})

	t.Run("unknown API path returns 404 JSON, not the SPA", func(t *testing.T) {
		t.Parallel()
		srv := newRouterServer(t)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodGet,
			Path:   "/api/nope",
		})
		testkit.AssertStatus(t, resp, body, http.StatusNotFound)
		testkit.AssertJSON(t, body, `{"code":"not_found"}`)
	})

	t.Run("panic in handler returns 500 unexpected_error", func(t *testing.T) {
		t.Parallel()
		clientApp, err := spa.Load()
		if err != nil {
			t.Fatal(err)
		}
		e := NewRouter(clientApp, nil)
		e.Logger = slog.New(slog.NewTextHandler(io.Discard, nil)) // keep the stack trace out of test output
		e.GET("/api/panic", func(c *echo.Context) error { panic("boom") })
		srv := httptest.NewServer(e)
		t.Cleanup(srv.Close)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: "/api/panic"})
		testkit.AssertStatus(t, resp, body, http.StatusInternalServerError)
		testkit.AssertJSON(t, body, `{"code":"unexpected_error"}`)
	})
}
