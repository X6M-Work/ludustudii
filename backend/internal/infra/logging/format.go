package logging

import "fmt"

// Format selects the output encoding of a logger built by New. The zero value
// is not a valid format.
type Format int

const (
	FormatJSON Format = iota + 1
	FormatText
)

// String implements fmt.Stringer. Unknown values print as "Format(n)".
func (f Format) String() string {
	switch f {
	case FormatJSON:
		return "json"
	case FormatText:
		return "text"
	}
	return fmt.Sprintf("Format(%d)", int(f))
}

// MarshalText implements encoding.TextMarshaler. It returns an error for
// values other than FormatJSON and FormatText.
func (f Format) MarshalText() ([]byte, error) {
	switch f {
	case FormatJSON, FormatText:
		return []byte(f.String()), nil
	}
	return nil, fmt.Errorf("unknown format %v", f)
}

// UnmarshalText implements encoding.TextUnmarshaler. It accepts "json" and
// "text".
func (f *Format) UnmarshalText(text []byte) error {
	switch string(text) {
	case "json":
		*f = FormatJSON
	case "text":
		*f = FormatText
	default:
		return fmt.Errorf("unknown format %q, want %q or %q", text, FormatJSON, FormatText)
	}
	return nil
}
