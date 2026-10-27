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
		{name: "NotFound", new: NotFound, code: "not_found"},
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

	first := validator.NewFieldError("name", "required", validator.ErrRequired)
	second := validator.NewFieldError("description", "max_length", validator.ErrMaxLength)
	third := validator.NewFieldError("id", "invalid_uuid", validator.ErrInvalidUUID7)
	unrelated := errors.New("unrelated")

	tests := []struct {
		name string
		err  error
		want []*validator.FieldError
	}{
		{name: "nil"},
		{name: "unrelated", err: unrelated},
		{name: "wrapped unrelated", err: fmt.Errorf("context: %w", unrelated)},
		{name: "field error", err: first, want: []*validator.FieldError{first}},
		{
			name: "multiply wrapped field error",
			err:  fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", first)),
			want: []*validator.FieldError{first},
		},
		{
			name: "joined field errors",
			err:  errors.Join(first, unrelated, second),
			want: []*validator.FieldError{first, second},
		},
		{
			name: "wrapped joined errors",
			err:  fmt.Errorf("context: %w", errors.Join(first, second)),
			want: []*validator.FieldError{first, second},
		},
		{
			name: "nested joined and wrapped errors",
			err: errors.Join(
				fmt.Errorf("context: %w", errors.Join(first, unrelated, second)),
				errors.Join(fmt.Errorf("context: %w", third)),
			),
			want: []*validator.FieldError{first, second, third},
		},
		{
			name: "multiple wrapped errors",
			err:  fmt.Errorf("first: %w; others: %w", first, errors.Join(second, third)),
			want: []*validator.FieldError{first, second, third},
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
