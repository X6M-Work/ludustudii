//go:build unit

package contract

import (
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/kit/sanitizer"
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
)

func FuzzString(f *testing.F) {
	f.Add("  Cafe\u0301 ")
	f.Fuzz(func(t *testing.T, s string) {
		// encoding/json never produces invalid UTF-8
		s = strings.ToValidUTF8(s, "\uFFFD")
		out := sanitizer.String(s)
		if err := validator.Trimmed("f", out); err != nil {
			t.Errorf("%q: %v", out, err)
		}
		if err := validator.Normalized("f", out); err != nil {
			t.Errorf("%q: %v", out, err)
		}
		if sanitizer.String(out) != out {
			t.Errorf("not idempotent: %q", out)
		}
	})
}
