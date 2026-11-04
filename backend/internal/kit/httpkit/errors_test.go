//go:build unit

package httpkit

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
)

func TestRequestErrorsRetainCause(t *testing.T) {
	t.Parallel()
	cause := errors.New("request failure")
	tests := []struct {
		name   string
		wrap   func(error) error
		status int
		code   string
	}{
		{"validation", Validation, http.StatusBadRequest, erresp.CodeValidation},
		{"precondition", PreconditionRequired, http.StatusPreconditionRequired, erresp.CodePreconditionRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := fmt.Errorf("request: %w", tt.wrap(cause))
			if !errors.Is(err, cause) {
				t.Fatalf("cause lost: %v", err)
			}
			if !strings.Contains(err.Error(), cause.Error()) {
				t.Fatalf("cause missing from error text: %v", err)
			}
			typed, ok := errors.AsType[*StatusError](err)
			if !ok || StatusCode(err) != tt.status || typed.Response.Code != tt.code {
				t.Fatalf("error: %#v, want %d %s", typed, tt.status, tt.code)
			}
		})
	}
}
