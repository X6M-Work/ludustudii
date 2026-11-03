//go:build unit

package app

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
	"github.com/apotourlyan/ludustudii/internal/testkit"
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
			err:        httpkit.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantJSON:   `{"code":"not_found"}`,
		},
		{
			name:       "GET 405 method_not_allowed",
			method:     http.MethodGet,
			err:        httpkit.ErrMethodNotAllowed,
			wantStatus: http.StatusMethodNotAllowed,
			wantJSON:   `{"code":"method_not_allowed"}`,
		},
		{
			name:       "GET 413 payload_too_large",
			method:     http.MethodGet,
			err:        httpkit.ErrPayloadTooLarge,
			wantStatus: http.StatusRequestEntityTooLarge,
			wantJSON:   `{"code":"payload_too_large"}`,
		},
		{
			name:       "POST 415 unsupported_media_type",
			method:     http.MethodPost,
			err:        httpkit.ErrUnsupportedMediaType,
			wantStatus: http.StatusUnsupportedMediaType,
			wantJSON:   `{"code":"unsupported_media_type"}`,
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
			err:        &httpkit.StatusError{Code: http.StatusUnauthorized},
			wantStatus: http.StatusUnauthorized,
			wantJSON:   `{"code":"request_error"}`,
		},
		{
			name:       "HEAD 404 returns empty body",
			method:     http.MethodHead,
			err:        httpkit.ErrNotFound,
			wantStatus: http.StatusNotFound,
		},
		{name: "typed server error becomes 500", method: http.MethodGet, err: &httpkit.StatusError{Code: http.StatusServiceUnavailable}, wantStatus: http.StatusInternalServerError, wantJSON: `{"code":"unexpected_error"}`, wantLog: true},
		{name: "wrapped routing error", method: http.MethodGet, err: fmt.Errorf("wrapped: %w", httpkit.ErrNotFound), wantStatus: http.StatusNotFound, wantJSON: `{"code":"not_found"}`},
		{name: "standard body-limit error", method: http.MethodPost, err: &http.MaxBytesError{Limit: MaxBodyBytes}, wantStatus: http.StatusRequestEntityTooLarge, wantJSON: `{"code":"payload_too_large"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logs bytes.Buffer
			log := slog.New(slog.NewTextHandler(&logs, nil))
			req := httptest.NewRequestWithContext(t.Context(), tt.method, "/api/test", nil)
			rec := httptest.NewRecorder()
			w := httpkit.NewResponseWriter(rec, req)

			httpErrorHandler(log, w, req, tt.err)

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

		log := slog.New(slog.DiscardHandler)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/test", nil)
		rec := httptest.NewRecorder()
		w := httpkit.NewResponseWriter(rec, req)

		if err := httpkit.Text(w, http.StatusOK, "x"); err != nil {
			t.Fatalf("write initial response: %v", err)
		}

		httpErrorHandler(log, w, req, errors.New("boom"))

		if rec.Code != http.StatusOK {
			t.Errorf("status: got %d, want %d", rec.Code, http.StatusOK)
		}
		if got := rec.Body.String(); got != "x" {
			t.Errorf("body: got %q, want %q", got, "x")
		}
	})
}
