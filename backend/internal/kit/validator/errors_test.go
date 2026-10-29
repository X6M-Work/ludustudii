//go:build unit

package validator

import (
	"errors"
	"fmt"
	"testing"
)

func TestNewFieldError(t *testing.T) {
	t.Parallel()

	err := errors.New("message")
	got := NewFieldError("field", "code", err)
	if got.Field != "field" || got.Code != "code" || got.Err != err {
		t.Errorf("field error: got %+v, want field/code/%v", got, err)
	}
}

func TestFieldError_Error(t *testing.T) {
	t.Parallel()

	fe := NewFieldError("field", "code", errors.New("message"))
	if got := fe.Error(); got != "message" {
		t.Errorf("error: got %q, want %q", got, "message")
	}
}

func TestFieldError_Unwrap(t *testing.T) {
	t.Parallel()

	fe := NewFieldError("field", CodeRequired, fmt.Errorf("%q: %w", "field", ErrRequired))
	if !errors.Is(fe, ErrRequired) {
		t.Errorf("errors.Is(fe, ErrRequired): got false, want true")
	}
	joined := errors.Join(errors.New("other"), fe)
	if !errors.Is(joined, ErrRequired) {
		t.Errorf("errors.Is(joined, ErrRequired): got false, want true")
	}
	if errors.Is(fe, ErrMaxLength) {
		t.Errorf("errors.Is(fe, ErrMaxLength): got true, want false")
	}
}
