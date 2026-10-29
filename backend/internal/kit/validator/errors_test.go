//go:build unit

package validator

import (
	"errors"
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
