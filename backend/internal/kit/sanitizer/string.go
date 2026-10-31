// Package sanitizer normalizes user input before it is validated.
package sanitizer

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

// String returns val in Unicode NFC with leading and trailing white space
// removed, so the result passes validator.Normalized and validator.Trimmed.
func String(val string) string {
	return strings.TrimSpace(norm.NFC.String(val))
}
