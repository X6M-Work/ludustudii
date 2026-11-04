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

	"github.com/apotourlyan/ludustudii/internal/kit"
	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
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
		{name: "storage not found", method: http.MethodGet, err: kit.ErrNotFound, wantStatus: http.StatusNotFound, wantJSON: `{"code":"not_found"}`},
		{name: "wrapped storage conflict", method: http.MethodPut, err: fmt.Errorf("update: %w", kit.ErrVersionConflict), wantStatus: http.StatusPreconditionFailed, wantJSON: `{"code":"precondition_failed"}`},
		{name: "parameter cause", method: http.MethodGet, err: httpkit.Param(errors.New("invalid UUID")), wantStatus: http.StatusBadRequest, wantJSON: `{"code":"param_error"}`},
		{name: "precondition cause", method: http.MethodPut, err: httpkit.PreconditionRequired(errors.New("invalid ETag")), wantStatus: http.StatusPreconditionRequired, wantJSON: `{"code":"precondition_required"}`},
		{name: "wrapped validation fields", method: http.MethodPost, err: fmt.Errorf("request: %w", httpkit.Validation(errors.Join(validator.Required("title", ""), validator.Required("description", "")))), wantStatus: http.StatusBadRequest, wantJSON: `{"code":"validation_error","errors":[{"field":"title","code":"required"},{"field":"description","code":"required"}]}`},
		{name: "binding cause", method: http.MethodPost, err: bindingError(t), wantStatus: http.StatusBadRequest, wantJSON: `{"code":"bind_error"}`},
		{name: "invalid success status becomes 500", method: http.MethodGet, err: &httpkit.StatusError{Code: http.StatusOK}, wantStatus: http.StatusInternalServerError, wantJSON: `{"code":"unexpected_error"}`, wantLog: true},
		{name: "typed server error becomes 500", method: http.MethodGet, err: &httpkit.StatusError{Code: http.StatusServiceUnavailable}, wantStatus: http.StatusInternalServerError, wantJSON: `{"code":"unexpected_error"}`, wantLog: true},
		{name: "wrapped routing error", method: http.MethodGet, err: fmt.Errorf("wrapped: %w", httpkit.ErrNotFound), wantStatus: http.StatusNotFound, wantJSON: `{"code":"not_found"}`},
		{name: "standard body-limit error", method: http.MethodPost, err: &http.MaxBytesError{Limit: MaxBodyBytes}, wantStatus: http.StatusRequestEntityTooLarge, wantJSON: `{"code":"payload_too_large"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logs bytes.Buffer
			log := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
			req := httptest.NewRequestWithContext(t.Context(), tt.method, "/api/test", nil)
			rec := httptest.NewRecorder()
			w := httpkit.NewResponseWriter(rec, req)

			httpErrorHandler(log, w, req, tt.err)

			if got := strings.Contains(logs.String(), "Unhandled error"); got != tt.wantLog {
				t.Errorf("logged: got %v, want %v\n%s", got, tt.wantLog, logs.String())
			}
			if tt.wantStatus < http.StatusInternalServerError {
				if !strings.Contains(logs.String(), "level=DEBUG") || !strings.Contains(logs.String(), "Request failed") {
					t.Errorf("missing debug log: %s", logs.String())
				}
				cause := tt.err
				if typed, ok := errors.AsType[*httpkit.StatusError](tt.err); ok && typed.Err != nil {
					cause = typed.Err
				}
				if !strings.Contains(logs.String(), fmt.Sprintf("%q", cause.Error())) && !strings.Contains(logs.String(), "error="+cause.Error()) {
					t.Errorf("log missing cause %q: %s", cause, logs.String())
				}
			}
			if rec.Code != tt.wantStatus {
				t.Fatalf("status: got %d, want %d\n%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type: got %q, want application/json", got)
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

func bindingError(t *testing.T) error {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/lcards", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	var target any
	return httpkit.BindJSON(req, &target)
}
