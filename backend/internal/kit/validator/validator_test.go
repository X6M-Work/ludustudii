//go:build unit

package validator

import (
	"errors"
	"strings"
	"testing"
)

func TestUUID7(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		val      string
		wantCode string
	}{
		{name: "lowercase", val: "0192f3a4-5b6c-7d8e-9f01-23456789abcd"},
		{name: "uppercase", val: "0192F3A4-5B6C-7D8E-9F01-23456789ABCD", wantCode: CodeInvalidUUID7},
		{name: "empty", val: "", wantCode: CodeInvalidUUID7},
		{name: "not a uuid", val: "not-a-uuid", wantCode: CodeInvalidUUID7},
		{name: "nil uuid", val: "00000000-0000-0000-0000-000000000000", wantCode: CodeInvalidUUID7},

		// version boundaries (first hex digit of the 3rd group)
		{name: "version 4", val: "9b2f6f4e-3c1a-4d2b-8e5f-7a6b5c4d3e2f", wantCode: CodeInvalidUUID7},
		{name: "version 6", val: "01a1730e-199d-6f9f-b59d-3bcd6f84066c", wantCode: CodeInvalidUUID7},
		{name: "version 7", val: "01a1730e-199d-7f9f-b59d-3bcd6f84066c"},
		{name: "version 8", val: "01a1730e-199d-8f9f-b59d-3bcd6f84066c", wantCode: CodeInvalidUUID7},

		// variant boundaries (first hex digit of the 4th group)
		{name: "variant 7 (NCS)", val: "01a1730e-199d-7f9f-759d-3bcd6f84066c", wantCode: CodeInvalidUUID7},
		{name: "variant 8 (lowest RFC)", val: "01a1730e-199d-7f9f-859d-3bcd6f84066c"},
		{name: "variant 9 (mid low RFC)", val: "01a1730e-199d-7f9f-959d-3bcd6f84066c"},
		{name: "variant a (mid high RFC)", val: "01a1730e-199d-7f9f-a59d-3bcd6f84066c"},
		{name: "variant b (highest RFC)", val: "01a1730e-199d-7f9f-b59d-3bcd6f84066c"},
		{name: "variant c (Microsoft)", val: "01a1730e-199d-7f9f-c59d-3bcd6f84066c", wantCode: CodeInvalidUUID7},
		{name: "variant d (Microsoft)", val: "01a1730e-199d-7f9f-d59d-3bcd6f84066c", wantCode: CodeInvalidUUID7},
		{name: "variant e (reserved)", val: "01a1730e-199d-7f9f-e59d-3bcd6f84066c", wantCode: CodeInvalidUUID7},
		{name: "variant f (reserved)", val: "01a1730e-199d-7f9f-f59d-3bcd6f84066c", wantCode: CodeInvalidUUID7},

		// parseable by uuid.Parse, but wrong database format
		{name: "braces", val: "{01a1730e-199d-7f9f-b59d-3bcd6f84066c}", wantCode: CodeInvalidUUID7},
		{name: "urn prefix", val: "urn:uuid:01a1730e-199d-7f9f-b59d-3bcd6f84066c", wantCode: CodeInvalidUUID7},
		{name: "no hyphens", val: "01a1730e199d7f9fb59d3bcd6f84066c", wantCode: CodeInvalidUUID7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := UUID7("id", tt.val)
			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			fe, ok := errors.AsType[*FieldError](err)
			if !ok {
				t.Fatalf("error: got %v, want *FieldError", err)
			}
			if fe.Field != "id" || fe.Code != tt.wantCode {
				t.Errorf("field error: got %s/%s, want id/%s", fe.Field, fe.Code, tt.wantCode)
			}
			if !errors.Is(fe.Err, ErrInvalidUUID7) {
				t.Errorf("err: got %v, want %v", fe.Err, ErrInvalidUUID7)
			}
		})
	}
}

func TestTrimmed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		val      string
		wantCode string
	}{
		{name: "trimmed", val: "Value"},
		{name: "empty", val: ""},
		{name: "inner whitespace", val: "A \t\nValue"},
		{name: "leading space", val: " Value", wantCode: CodeUntrimmed},
		{name: "trailing space", val: "Value ", wantCode: CodeUntrimmed},
		{name: "leading tab", val: "\tValue", wantCode: CodeUntrimmed},
		{name: "trailing tab", val: "Value\t", wantCode: CodeUntrimmed},
		{name: "leading newline", val: "\nValue", wantCode: CodeUntrimmed},
		{name: "trailing newline", val: "Value\n", wantCode: CodeUntrimmed},
		{name: "leading carriage return", val: "\rValue", wantCode: CodeUntrimmed},
		{name: "trailing carriage return", val: "Value\r", wantCode: CodeUntrimmed},
		{name: "leading unicode whitespace", val: "\u00A0Value", wantCode: CodeUntrimmed},
		{name: "trailing unicode whitespace", val: "Value\u00A0", wantCode: CodeUntrimmed},
		{name: "whitespace only", val: " \t\r\n\u00A0\u2003", wantCode: CodeUntrimmed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Trimmed("field", tt.val)
			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			fe, ok := errors.AsType[*FieldError](err)
			if !ok {
				t.Fatalf("error: got %v, want *FieldError", err)
			}
			if fe.Field != "field" || fe.Code != tt.wantCode {
				t.Errorf("field error: got %s/%s, want field/%s", fe.Field, fe.Code, tt.wantCode)
			}
			if !errors.Is(fe.Err, ErrUntrimmed) {
				t.Errorf("err: got %v, want %v", fe.Err, ErrUntrimmed)
			}
		})
	}
}

func TestNormalized(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		val      string
		wantCode string
	}{
		{name: "ascii", val: "Value"},
		{name: "empty", val: ""},
		{name: "precomposed", val: "Caf\u00E9"},
		{name: "decomposed", val: "Cafe\u0301", wantCode: CodeNormalized},
		{name: "partially decomposed", val: "\u00E9e\u0301", wantCode: CodeNormalized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Normalized("field", tt.val)
			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			fe, ok := errors.AsType[*FieldError](err)
			if !ok {
				t.Fatalf("error: got %v, want *FieldError", err)
			}
			if fe.Field != "field" || fe.Code != tt.wantCode {
				t.Errorf("field error: got %s/%s, want field/%s", fe.Field, fe.Code, tt.wantCode)
			}
			if !errors.Is(fe.Err, ErrNormalized) {
				t.Errorf("err: got %v, want %v", fe.Err, ErrNormalized)
			}
		})
	}
}

func TestRequired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		val      string
		wantCode string
	}{
		{name: "value", val: "Value"},
		{name: "whitespace", val: " "},
		{name: "empty", val: "", wantCode: CodeRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Required("field", tt.val)
			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			fe, ok := errors.AsType[*FieldError](err)
			if !ok {
				t.Fatalf("error: got %v, want *FieldError", err)
			}
			if fe.Field != "field" || fe.Code != tt.wantCode {
				t.Errorf("field error: got %s/%s, want field/%s", fe.Field, fe.Code, tt.wantCode)
			}
			if !errors.Is(fe.Err, ErrRequired) {
				t.Errorf("err: got %v, want %v", fe.Err, ErrRequired)
			}
		})
	}
}

func TestMaxLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		val      string
		max      int
		wantCode string
	}{
		{name: "empty", val: "", max: 5},
		{name: "at max", val: "abcde", max: 5},
		{name: "at max multi-byte", val: strings.Repeat("\u00E9", 5), max: 5},
		{name: "over max", val: "abcdef", max: 5, wantCode: CodeMaxLength},
		{name: "over max multi-byte", val: strings.Repeat("\u00E9", 6), max: 5, wantCode: CodeMaxLength},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := MaxLength("field", tt.val, tt.max)
			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			fe, ok := errors.AsType[*FieldError](err)
			if !ok {
				t.Fatalf("error: got %v, want *FieldError", err)
			}
			if fe.Field != "field" || fe.Code != tt.wantCode {
				t.Errorf("field error: got %s/%s, want field/%s", fe.Field, fe.Code, tt.wantCode)
			}
			if !errors.Is(fe.Err, ErrMaxLength) {
				t.Errorf("err: got %v, want %v", fe.Err, ErrMaxLength)
			}
		})
	}
}
