//go:build unit

package httpkit

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
)

func TestBindQueryInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		query        string
		defaultValue int
		want         int
		wantErr      error
		wantErrText  string
	}{
		{name: "missing", defaultValue: 20, want: 20},
		{name: "empty", query: "?size=", defaultValue: 20, want: 20},
		{name: "zero default", want: 0},
		{name: "negative default", defaultValue: -1, want: -1},
		{name: "explicit", query: "?size=5", defaultValue: 20, want: 5},
		{name: "zero", query: "?size=0", defaultValue: 20, want: 0},
		{name: "negative", query: "?size=-1", defaultValue: 20, want: -1},
		{name: "unrelated parameter", query: "?limit=5", defaultValue: 20, want: 20},
		{name: "letters", query: "?size=abc", wantErr: strconv.ErrSyntax},
		{name: "fraction", query: "?size=1.5", wantErr: strconv.ErrSyntax},
		{name: "leading zero", query: "?size=05", wantErrText: "not canonical:"},
		{name: "negative leading zero", query: "?size=-05", wantErrText: "not canonical:"},
		{name: "negative zero", query: "?size=-0", wantErrText: "not canonical:"},
		{name: "plus sign", query: "?size=%2B5", wantErrText: "not canonical:"},
		{name: "leading whitespace", query: "?size=%205", wantErr: strconv.ErrSyntax},
		{name: "trailing whitespace", query: "?size=5%20", wantErr: strconv.ErrSyntax},
		{name: "overflow", query: "?size=99999999999999999999999999999", wantErr: strconv.ErrRange},
		{name: "underflow", query: "?size=-99999999999999999999999999999", wantErr: strconv.ErrRange},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/"+tt.query, nil)
			got, err := BindQueryInt(r, "size", tt.defaultValue)
			if tt.wantErr != nil || tt.wantErrText != "" {
				typed, ok := errors.AsType[*StatusError](err)
				if !ok || typed.Code != http.StatusBadRequest || typed.Response.Code != erresp.CodeBinding || typed.Err == nil {
					t.Fatalf("expected 400 bind_error, got %v", err)
				}
			}
			if tt.wantErrText != "" {
				if !strings.Contains(errors.Unwrap(err).Error(), tt.wantErrText) {
					t.Fatalf("error: got %v, want %s", err, tt.wantErrText)
				}
			} else if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error: got %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("value: got %d, want %d", got, tt.want)
			}
		})
	}
}
