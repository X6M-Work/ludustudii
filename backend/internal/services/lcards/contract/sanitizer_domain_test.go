//go:build unit

package contract

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/apotourlyan/ludustudii/internal/kit/sanitizer"
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
)

var sanitizedFields = []struct {
	field    string
	validate func(string) error
	max      int
}{
	{"title", domain.ValidateTitle, domain.TitleMaxLength},
	{"description", domain.ValidateDescription, domain.DescriptionMaxLength},
}

func FuzzValidateSanitized(f *testing.F) {
	f.Add("")
	f.Add("   ")
	f.Add("\u00A0x\u00A0")
	f.Add("  Cafe\u0301 ")
	f.Add(strings.Repeat("\u00E9", 200))
	f.Add(strings.Repeat("\u00E9", 201))
	// 400 runes before NFC, 200 after
	f.Add(strings.Repeat("e\u0301", 200))
	f.Add(strings.Repeat("a", 2000))
	f.Add(strings.Repeat("a", 2001))
	f.Fuzz(func(t *testing.T, in string) {
		// encoding/json never produces invalid UTF-8
		in = strings.ToValidUTF8(in, "\uFFFD")
		out := sanitizer.String(in)
		for _, tc := range sanitizedFields {
			checkSanitized(t, tc.field, tc.validate, tc.max, in, out)
		}
	})
}

func checkSanitized(
	t *testing.T,
	field string,
	validate func(string) error,
	max int,
	in, out string,
) {
	t.Helper()

	var wantCode string
	switch {
	case out == "":
		wantCode = validator.CodeRequired
	case utf8.RuneCountInString(out) > max:
		wantCode = validator.CodeMaxLength
	}

	err := validate(out)
	if wantCode == "" {
		if err != nil {
			t.Errorf("%s: input %q sanitized %q: got %v, want nil",
				field, in, out, err)
		}
		return
	}

	fe, ok := errors.AsType[*validator.FieldError](err)
	if !ok {
		t.Errorf("%s: input %q sanitized %q: got %v, want *validator.FieldError",
			field, in, out, err)
		return
	}
	if fe.Field != field {
		t.Errorf("%s: input %q sanitized %q: field got %q, want %q",
			field, in, out, fe.Field, field)
	}
	if fe.Code != wantCode {
		t.Errorf("%s: input %q sanitized %q: code got %q, want %q",
			field, in, out, fe.Code, wantCode)
	}
}
