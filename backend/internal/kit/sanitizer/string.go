package sanitizer

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

func String(val string) string {
	return strings.TrimSpace(norm.NFC.String(val))
}
