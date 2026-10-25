package validator

import (
	"fmt"
	"unicode/utf8"

	"github.com/google/uuid"
)

func UUID7(field, val string) error {
	id, err := uuid.Parse(val)
	if err != nil || id.Version() != 7 || id.Variant() != uuid.RFC4122 {
		return fmt.Errorf("%s %w", field, ErrInvalidUUID7)
	}
	return nil
}

func Required(field, val string) error {
	if val == "" {
		return fmt.Errorf("%s %w", field, ErrRequired)
	}
	return nil
}

func MaxLength(field, val string, max int) error {
	curr := utf8.RuneCountInString(val)
	if curr > max {
		return fmt.Errorf("%s %d/%d: %w", field, curr, max, ErrMaxLength)
	}
	return nil
}
