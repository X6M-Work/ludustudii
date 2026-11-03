//go:build unit

package httpkit

import (
	"net/http"
	"net/http/httptest"
	"slices"
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
