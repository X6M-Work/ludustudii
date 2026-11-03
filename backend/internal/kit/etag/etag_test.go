//go:build unit

package etag

import (
	"errors"
	"math"
	"testing"
)

func TestFormat(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		version int64
		want    string
	}{
		{1, `"1"`}, {3, `"3"`}, {math.MaxInt64, `"9223372036854775807"`},
	} {
		if got := Format(tt.version); got != tt.want {
			t.Errorf("Format(%d) = %q, want %q", tt.version, got, tt.want)
		}
	}
}

func TestParseVersion(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		value string
		want  int64
	}{
		{`"1"`, 1},
		{`"3"`, 3},
		{`"01"`, 1},
		{`"9223372036854775807"`, math.MaxInt64},
		{"", 0},
		{"*", 0},
		{`W/"1"`, 0},
		{`"1", "2"`, 0},
		{"1", 0},
		{`"abc"`, 0},
		{`""`, 0},
		{`"0"`, 0},
		{`"-1"`, 0},
		{`"+1"`, 0},
		{`"9223372036854775808"`, 0},
		{` "1"`, 0},
		{`"1" `, 0},
		{`"1.0"`, 0},
	} {
		got, err := ParseVersion(tt.value)
		if tt.want == 0 {
			if !errors.Is(err, ErrInvalid) {
				t.Errorf("ParseVersion(%q): got %v, want ErrInvalid", tt.value, err)
			}
		} else if err != nil || got != tt.want {
			t.Errorf("ParseVersion(%q) = %d, %v; want %d", tt.value, got, err, tt.want)
		}
	}
}

func FuzzRoundTrip(f *testing.F) {
	f.Add(int64(1))
	f.Add(int64(math.MaxInt64))
	f.Fuzz(func(t *testing.T, version int64) {
		if version < 1 {
			t.Skip()
		}
		got, err := ParseVersion(Format(version))
		if err != nil || got != version {
			t.Fatalf("round trip %d = %d, %v", version, got, err)
		}
	})
}
