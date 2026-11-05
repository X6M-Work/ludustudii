//go:build unit

package httpkit

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"testing"
)

func TestChain(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		count int
		stop  bool
		want  []string
	}{
		{name: "empty", want: []string{"handler"}},
		{name: "request and response order", count: 2, want: []string{"outer before", "inner before", "handler", "inner after", "outer after"}},
		{name: "short circuit", count: 2, stop: true, want: []string{"outer before", "outer after"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var calls []string
			middleware := func(name string, stop bool) Middleware {
				return func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls = append(calls, name+" before")
						if !stop {
							next.ServeHTTP(w, r)
						}
						calls = append(calls, name+" after")
					})
				}
			}
			var middlewares []Middleware
			if tc.count != 0 {
				middlewares = []Middleware{middleware("outer", tc.stop), middleware("inner", false)}
			}
			chain := Chain(middlewares...)
			for range 2 {
				calls = nil
				handler := chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls = append(calls, "handler") }))
				handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
				if !slices.Equal(calls, tc.want) {
					t.Fatalf("calls: %v, want %v", calls, tc.want)
				}
			}
		})
	}
}

func TestRequestLogger_QuietPaths(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		path   string
		status int
		level  string
	}{
		{"/operational", 200, "DEBUG"},
		{"/operational", 301, "DEBUG"},
		{"/operational", 405, "DEBUG"},
		{"/operational", 503, "DEBUG"},
		{"/ordinary", 200, "INFO"},
		{"/ordinary", 405, "WARN"},
		{"/ordinary", 503, "ERROR"},
		{"/operational/child", 503, "ERROR"},
	} {
		t.Run(tc.path+"/"+strconv.Itoa(tc.status), func(t *testing.T) {
			t.Parallel()
			var logs bytes.Buffer
			log := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
			handler := RequestLogger(log, RequestLoggerOptions{QuietPaths: map[string]struct{}{"/operational": {}}})(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(tc.status) }))
			r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, nil)
			handler.ServeHTTP(NewResponseWriter(httptest.NewRecorder(), r), r)
			var record map[string]any
			if err := json.Unmarshal(logs.Bytes(), &record); err != nil {
				t.Fatalf("expected one JSON log record: %v", err)
			}
			if record["msg"] != "request" || record["level"] != tc.level || record["status"] != float64(tc.status) {
				t.Fatalf("record: %v, want status %d level %s", record, tc.status, tc.level)
			}
		})
	}
}
