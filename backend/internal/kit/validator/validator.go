package validator

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"
)

const (
	CodeInvalidUUID7   = "invalid_uuid7"
	CodeRequired       = "required"
	CodeMaxLength      = "max_length"
	CodeUntrimmed      = "untrimmed"
	CodeNormalized     = "not_normalized"
	CodeInvalidInteger = "invalid_integer"
	CodeOutOfRange     = "out_of_range"
)

func UUID7(field, val string) error {
	id, err := uuid.Parse(val)
	if err != nil ||
		id.String() != val ||
		id.Version() != 7 ||
		id.Variant() != uuid.RFC4122 {
		return NewFieldError(field, CodeInvalidUUID7,
			fmt.Errorf("%q: %w", field, ErrInvalidUUID7))
	}
	return nil
}

func Trimmed(field, val string) error {
	if val != strings.TrimSpace(val) {
		return NewFieldError(field, CodeUntrimmed,
			fmt.Errorf("%q: %w", field, ErrUntrimmed))
	}
	return nil
}

func Normalized(field, val string) error {
	if val != norm.NFC.String(val) {
		return NewFieldError(field, CodeNormalized,
			fmt.Errorf("%q: %w", field, ErrNormalized))
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

// Integer accepts only the canonical base-10 form, so "+5", "05" and " 5" fail.
func Integer(field, val string) error {
	n, err := strconv.Atoi(val)
	if err != nil || strconv.Itoa(n) != val {
		return NewFieldError(field, CodeInvalidInteger,
			fmt.Errorf("%q: %w", field, ErrInvalidInteger))
	}
	return nil
}

func IntRange(field string, val, min, max int) error {
	if val < min || val > max {
		return NewFieldError(field, CodeOutOfRange,
			fmt.Errorf("%q %d not in %d..%d: %w", field, val, min, max, ErrOutOfRange))
	}
	return nil
}
