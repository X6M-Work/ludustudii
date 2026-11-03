//go:build unit

package httpkit

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBodyLimit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, body string
		length     int64
		status     int
		entered    bool
	}{
		{"known at limit", `"abcdef"`, 8, 200, true},
		{"known over limit", `"abcdefg"`, 9, 413, false},
		{"unknown at limit", `"abcdef"`, -1, 200, true},
		{"unknown over limit", `"abcdefg"`, -1, 413, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			entered := false
			onError := func(w http.ResponseWriter, _ *http.Request, err error) {
				if tc.length == -1 {
					if _, ok := errors.AsType[*http.MaxBytesError](err); !ok {
						t.Errorf("error: %T, want MaxBytesError", err)
					}
				}
				w.WriteHeader(StatusCode(err))
			}
			next := Adapt(func(w http.ResponseWriter, r *http.Request) error {
				entered = true
				var target string
				return BindJSON(w, r, &target)
			}, onError)
			r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(tc.body))
			r.ContentLength = tc.length
			r.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			BodyLimit(8, onError)(next).ServeHTTP(rec, r)
			if rec.Code != tc.status || entered != tc.entered {
				t.Fatalf("status/entered: %d/%v, want %d/%v", rec.Code, entered, tc.status, tc.entered)
			}
		})
	}
}

type controllerWriter struct {
	*httptest.ResponseRecorder
	deadline time.Time
}

func (w *controllerWriter) SetWriteDeadline(deadline time.Time) error {
	w.deadline = deadline
	return nil
}

func TestResponseWriter_ControllerAndCommit(t *testing.T) {
	t.Parallel()
	base := &controllerWriter{ResponseRecorder: httptest.NewRecorder()}
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	w := NewResponseWriter(base, r)
	controller := http.NewResponseController(w)
	deadline := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	if err := controller.SetWriteDeadline(deadline); err != nil {
		t.Fatal(err)
	}
	if base.deadline != deadline || w.Unwrap() != base {
		t.Fatal("controller did not unwrap writer")
	}
	if err := controller.Flush(); err != nil {
		t.Fatal(err)
	}
	if !w.Committed || !base.Flushed || w.Status != http.StatusOK {
		t.Fatal("flush did not commit response")
	}
	w.WriteHeader(http.StatusInternalServerError)
	if w.Status != http.StatusOK || base.Code != http.StatusOK {
		t.Fatal("second header changed status")
	}
}

func TestRecover_AbortHandler(t *testing.T) {
	t.Parallel()
	defer func() {
		if got := recover(); got != http.ErrAbortHandler { //nolint:errorlint // The HTTP server recognizes only the exact sentinel.
			t.Fatalf("panic: %v, want ErrAbortHandler", got)
		}
	}()
	h := Recover(func(http.ResponseWriter, *http.Request, error) { t.Fatal("abort was reported as an error") })(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
}
