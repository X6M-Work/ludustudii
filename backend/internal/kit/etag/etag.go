// Package etag formats and parses HTTP entity tags containing resource versions.
package etag

import (
	"errors"
	"strconv"
)

// ErrInvalid indicates that If-Match is not one strong positive int64 tag.
var ErrInvalid = errors.New("expected one strong positive integer ETag")

// Format returns the strong ETag for version.
func Format(version int64) string {
	return `"` + strconv.FormatInt(version, 10) + `"`
}

// ParseVersion accepts exactly one quoted positive decimal int64.
func ParseVersion(value string) (int64, error) {
	if len(value) < 3 || value[0] != '"' || value[len(value)-1] != '"' {
		return 0, ErrInvalid
	}
	digits := value[1 : len(value)-1]
	for _, c := range digits {
		if c < '0' || c > '9' {
			return 0, ErrInvalid
		}
	}
	version, err := strconv.ParseInt(digits, 10, 64)
	if err != nil || version < 1 {
		return 0, ErrInvalid
	}
	return version, nil
}
