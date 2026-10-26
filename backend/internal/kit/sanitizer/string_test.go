package sanitizer

import "testing"

func TestString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  string
		want string
	}{
		{name: "empty", val: "", want: ""},
		{name: "clean", val: "Value", want: "Value"},
		{name: "leading and trailing spaces", val: "  Value  ", want: "Value"},
		{name: "tabs and newlines", val: "\t\nValue\n\t", want: "Value"},
		{name: "unicode whitespace", val: "\u00A0Value\u2003", want: "Value"},
		{name: "whitespace only", val: " \t\r\n ", want: ""},
		{name: "inner whitespace kept", val: " A  \n Value ", want: "A  \n Value"},
		{name: "precomposed kept", val: "Caf\u00E9", want: "Caf\u00E9"},
		{name: "decomposed composed", val: "Cafe\u0301", want: "Caf\u00E9"},
		{name: "decomposed and untrimmed", val: " Cafe\u0301 ", want: "Caf\u00E9"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := String(tt.val)
			if got != tt.want {
				t.Fatalf("string: got %q, want %q", got, tt.want)
			}
			if again := String(got); again != got {
				t.Errorf("not idempotent: got %q, want %q", again, got)
			}
		})
	}
}
