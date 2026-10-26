package validator

import (
	"fmt"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	CodeInvalidUUID7 = "invalid_uuid7"
	CodeRequired     = "required"
	CodeMaxLength    = "max_length"
)

func UUID7(field, val string) error {
	id, err := uuid.Parse(val)
	if err != nil || id.Version() != 7 || id.Variant() != uuid.RFC4122 {
		return NewFieldError(field, CodeInvalidUUID7,
			fmt.Errorf("%q: %w", field, ErrInvalidUUID7))
	}
	return nil
}

func Required(field, val string) error {
	if val == "" {
		return NewFieldError(field, CodeRequired,
			fmt.Errorf("%q: %w", field, ErrRequired))
	}
	return nil
}

func MaxLength(field, val string, max int) error {
	curr := utf8.RuneCountInString(val)
	if curr > max {
		return NewFieldError(field, CodeMaxLength,
			fmt.Errorf("%q %d/%d: %w", field, curr, max, ErrMaxLength))
	}
	return nil
}
