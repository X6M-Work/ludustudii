package app

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/testkit"
	"github.com/labstack/echo/v5"
)

func TestHTTPErrorHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		method     string
		err        error
		wantStatus int
		wantJSON   string
		wantLog    bool
	}{
		{
			name:       "GET 404 not_found",
			method:     http.MethodGet,
			err:        echo.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantJSON:   `{"code":"not_found"}`,
		},
		{
			name:       "GET 405 method_not_allowed",
			method:     http.MethodGet,
			err:        echo.ErrMethodNotAllowed,
			wantStatus: http.StatusMethodNotAllowed,
			wantJSON:   `{"code":"method_not_allowed"}`,
		},
		{
			name:       "GET 413 payload_too_large",
			method:     http.MethodGet,
			err:        echo.ErrStatusRequestEntityTooLarge,
			wantStatus: http.StatusRequestEntityTooLarge,
			wantJSON:   `{"code":"payload_too_large"}`,
		},
		{
			name:       "GET plain error becomes 500 unexpected_error",
			method:     http.MethodGet,
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantJSON:   `{"code":"unexpected_error"}`,
			wantLog:    true,
		},
		{
			name:       "GET 401 keeps status as request_error",
			method:     http.MethodGet,
			err:        echo.ErrUnauthorized,
			wantStatus: http.StatusUnauthorized,
			wantJSON:   `{"code":"request_error"}`,
		},
		{
			name:       "HEAD 404 returns empty body",
			method:     http.MethodHead,
			err:        echo.ErrNotFound,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logs bytes.Buffer
			e := echo.NewWithConfig(echo.Config{
				Logger: slog.New(slog.NewTextHandler(&logs, nil)),
			})
			req := httptest.NewRequest(tt.method, "/api/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			httpErrorHandler(c, tt.err)

			if got := strings.Contains(logs.String(), "Unhandled error"); got != tt.wantLog {
				t.Errorf("logged: got %v, want %v\n%s", got, tt.wantLog, logs.String())
			}
			if rec.Code != tt.wantStatus {
				t.Fatalf("status: got %d, want %d\n%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantJSON == "" {
				if rec.Body.Len() != 0 {
					t.Errorf("body: got %q, want empty", rec.Body.String())
				}
				return
			}
			testkit.AssertJSON(t, rec.Body.Bytes(), tt.wantJSON)
		})
	}

	t.Run("committed response is left alone", func(t *testing.T) {
		t.Parallel()

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := c.String(http.StatusOK, "x"); err != nil {
			t.Fatalf("write initial response: %v", err)
		}

		httpErrorHandler(c, errors.New("boom"))

		if rec.Code != http.StatusOK {
			t.Errorf("status: got %d, want %d", rec.Code, http.StatusOK)
		}
		if got := rec.Body.String(); got != "x" {
			t.Errorf("body: got %q, want %q", got, "x")
		}
	})
}
