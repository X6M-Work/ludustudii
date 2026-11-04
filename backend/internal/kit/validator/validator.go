// Package validator provides field-level checks for request input. Each check
// returns nil or a *FieldError that names the field, carries a Code for the
// client and wraps one of the Err sentinels.
package validator

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"
)

// Machine-readable codes sent in FieldError.Code.
const (
	CodeInvalidUUID7   = "invalid_uuid7"
	CodeRequired       = "required"
	CodeMaxLength      = "max_length"
	CodeUntrimmed      = "untrimmed"
	CodeNormalized     = "not_normalized"
	CodeOutOfRange     = "out_of_range"
	CodeInvalidInteger = "invalid_integer"
)

// EmptyOrUUID7 accepts an empty string or a UUID7 in canonical form.
func EmptyOrUUID7(field, val string) error {
	if val == "" {
		return nil
	}
	return UUID7(field, val)
}

// UUID7 checks that val is a UUID version 7 (RFC 9562 variant) in canonical
// form: lowercase and hyphenated, without braces or a urn:uuid: prefix.
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

// Trimmed checks that val has no leading or trailing white space.
func Trimmed(field, val string) error {
	if val != strings.TrimSpace(val) {
		return NewFieldError(field, CodeUntrimmed,
			fmt.Errorf("%q: %w", field, ErrUntrimmed))
	}
	return nil
}

// Normalized checks that val is in Unicode NFC.
func Normalized(field, val string) error {
	if val != norm.NFC.String(val) {
		return NewFieldError(field, CodeNormalized,
			fmt.Errorf("%q: %w", field, ErrNormalized))
	}
	return nil
}

// Required checks that val is not empty.
func Required(field, val string) error {
	if val == "" {
		return NewFieldError(field, CodeRequired,
			fmt.Errorf("%q: %w", field, ErrRequired))
	}
	return nil
}

// MaxLength checks that val has at most max characters, counted in runes.
func MaxLength(field, val string, max int) error {
	curr := utf8.RuneCountInString(val)
	if curr > max {
		return NewFieldError(field, CodeMaxLength,
			fmt.Errorf("%q %d/%d: %w", field, curr, max, ErrMaxLength))
	}
	return nil
}

// IntRange checks that min <= val <= max.
func IntRange(field string, val, min, max int) error {
	if val < min || val > max {
		return NewFieldError(field, CodeOutOfRange,
			fmt.Errorf("%q %d not in %d..%d: %w", field, val, min, max, ErrOutOfRange))
	}
	return nil
}
