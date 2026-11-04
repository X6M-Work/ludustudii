//go:build unit

package validator

import (
	"errors"
	"testing"
)

func TestValidateMaxLengthText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		field    string
		val      string
		max      int
		wantCode string
		wantErr  error
	}{
		{name: "valid", field: "title", val: "Title", max: 10},
		{name: "single rune", field: "title", val: "a", max: 1},
		{name: "at max length", field: "title", val: "abcde", max: 5},
		{name: "at max length in runes", field: "description", val: "Caf\u00E9", max: 4},
		{name: "inner whitespace", field: "description", val: "a \t\nb", max: 5},
		{name: "untrimmed", field: "title", val: " Title ", max: 10, wantCode: CodeUntrimmed, wantErr: ErrUntrimmed},
		{name: "not normalized", field: "description", val: "Cafe\u0301", max: 10, wantCode: CodeNormalized, wantErr: ErrNormalized},
		{name: "empty", field: "body", max: 10, wantCode: CodeRequired, wantErr: ErrRequired},
		{name: "over max length", field: "body", val: "abcdef", max: 5, wantCode: CodeMaxLength, wantErr: ErrMaxLength},
		{name: "over max length in runes", field: "description", val: "\u00E9\u00E9", max: 1, wantCode: CodeMaxLength, wantErr: ErrMaxLength},
		{name: "zero max rejects text", field: "body", val: "a", wantCode: CodeMaxLength, wantErr: ErrMaxLength},
		{name: "zero max still requires text", field: "body", wantCode: CodeRequired, wantErr: ErrRequired},
		{name: "trim check precedes normalization and length", field: "body", val: " e\u0301 ", max: 1, wantCode: CodeUntrimmed, wantErr: ErrUntrimmed},
		{name: "normalization check precedes length", field: "body", val: "e\u0301", max: 1, wantCode: CodeNormalized, wantErr: ErrNormalized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateMaxLengthText(tt.field, tt.val, tt.max)
			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			fe, ok := errors.AsType[*FieldError](err)
			if !ok {
				t.Fatalf("error: got %v, want *FieldError", err)
			}
			if fe.Field != tt.field || fe.Code != tt.wantCode {
				t.Errorf("field error: got %s/%s, want %s/%s", fe.Field, fe.Code, tt.field, tt.wantCode)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("err: got %v, want %v", err, tt.wantErr)
			}
		})
	}
}
