//go:build unit

package logging

import "testing"

func TestFormat_Text(t *testing.T) {
	t.Parallel()

	tests := []struct {
		text   string
		format Format
	}{
		{text: "json", format: FormatJSON},
		{text: "text", format: FormatText},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			t.Parallel()

			if got := tt.format.String(); got != tt.text {
				t.Errorf("String: got %q, want %q", got, tt.text)
			}

			b, err := tt.format.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText: %v", err)
			}
			if string(b) != tt.text {
				t.Errorf("MarshalText: got %q, want %q", b, tt.text)
			}

			var got Format
			if err := got.UnmarshalText([]byte(tt.text)); err != nil {
				t.Fatalf("UnmarshalText: %v", err)
			}
			if got != tt.format {
				t.Errorf("UnmarshalText: got %v, want %v", got, tt.format)
			}
		})
	}

	t.Run("unknown text", func(t *testing.T) {
		t.Parallel()

		for _, text := range []string{"", "JSON", "xml"} {
			f := FormatText
			if err := f.UnmarshalText([]byte(text)); err == nil {
				t.Errorf("UnmarshalText(%q): expected error", text)
			}
			if f != FormatText {
				t.Errorf("UnmarshalText(%q): changed value to %v", text, f)
			}
		}
	})

	t.Run("unknown format", func(t *testing.T) {
		t.Parallel()

		if got := Format(0).String(); got != "Format(0)" {
			t.Errorf("String: got %q, want %q", got, "Format(0)")
		}
		if _, err := Format(0).MarshalText(); err == nil {
			t.Error("MarshalText: expected error")
		}
	})
}
