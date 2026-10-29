//go:build unit

package erresp

import (
	"errors"
	"fmt"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/kit/validator"
)

func TestConstructors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		new  func() ErrorResponse
		code string
	}{
		{name: "New", new: func() ErrorResponse { return New("custom_error") }, code: "custom_error"},
		{name: "Binding", new: Binding, code: "bind_error"},
		{name: "MethodNotAllowed", new: MethodNotAllowed, code: "method_not_allowed"},
		{name: "NotFound", new: NotFound, code: "not_found"},
		{name: "PayloadTooLarge", new: PayloadTooLarge, code: "payload_too_large"},
		{name: "Unexpected", new: Unexpected, code: "unexpected_error"},
		{name: "Param", new: Param, code: "param_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.new()
			if got.Code != tt.code {
				t.Errorf("code: got %q, want %q", got.Code, tt.code)
			}
			if got.Errors != nil {
				t.Errorf("errors: got %v, want nil", got.Errors)
			}
		})
	}
}

func TestValidation(t *testing.T) {
	t.Parallel()

	a := validator.NewFieldError("name", validator.CodeRequired, validator.ErrRequired)
	b := validator.NewFieldError("description", validator.CodeMaxLength, validator.ErrMaxLength)
	c := validator.NewFieldError("id", validator.CodeInvalidUUID7, validator.ErrInvalidUUID7)
	unrelated := errors.New("unrelated")

	tests := []struct {
		name string
		err  error
		want []*validator.FieldError
	}{
		{name: "nil"},
		{name: "unrelated", err: unrelated},
		{name: "wrapped unrelated", err: fmt.Errorf("context: %w", unrelated)},
		{name: "field error", err: a, want: []*validator.FieldError{a}},
		{
			name: "multiply wrapped field error",
			err:  fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", a)),
			want: []*validator.FieldError{a},
		},
		{
			name: "joined field errors",
			err:  errors.Join(a, unrelated, b),
			want: []*validator.FieldError{a, b},
		},
		{
			name: "wrapped joined errors",
			err:  fmt.Errorf("context: %w", errors.Join(a, b)),
			want: []*validator.FieldError{a, b},
		},
		{
			name: "nested joined and wrapped errors",
			err: errors.Join(
				fmt.Errorf("context: %w", errors.Join(a, unrelated, b)),
				errors.Join(fmt.Errorf("context: %w", c)),
			),
			want: []*validator.FieldError{a, b, c},
		},
		{
			name: "multiple wrapped errors",
			err:  fmt.Errorf("first: %w; others: %w", a, errors.Join(b, c)),
			want: []*validator.FieldError{a, b, c},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Validation(tt.err)
			if got.Code != CodeValidation {
				t.Errorf("code: got %q, want %q", got.Code, CodeValidation)
			}
			if len(got.Errors) != len(tt.want) {
				t.Fatalf("errors: got %v, want %v", got.Errors, tt.want)
			}
			for i, want := range tt.want {
				if got.Errors[i] != want {
					t.Errorf("errors[%d]: got %+v, want %+v", i, got.Errors[i], want)
				}
			}
		})
	}
}
