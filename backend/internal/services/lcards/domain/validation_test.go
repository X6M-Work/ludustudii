//go:build unit

package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/kit/validator"
)

func TestValidateTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		val      string
		wantCode string
	}{
		{name: "valid", val: "Title"},
		{name: "at min length", val: strings.Repeat("a", 1)},
		{name: "at max length", val: strings.Repeat("\u00E9", TitleMaxLength)},
		{name: "normalized", val: "Caf\u00E9"},
		{name: "untrimmed", val: " Title ", wantCode: validator.CodeUntrimmed},
		{name: "not normalized", val: "Cafe\u0301", wantCode: validator.CodeNormalized},
		{name: "empty", val: "", wantCode: validator.CodeRequired},
		{name: "over max length", val: strings.Repeat("\u00E9", TitleMaxLength+1), wantCode: validator.CodeMaxLength},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateTitle(tt.val)
			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			fe, ok := errors.AsType[*validator.FieldError](err)
			if !ok {
				t.Fatalf("error: got %v, want *validator.FieldError", err)
			}
			if fe.Field != "title" || fe.Code != tt.wantCode {
				t.Errorf("field error: got %s/%s, want title/%s", fe.Field, fe.Code, tt.wantCode)
			}
		})
	}
}

func TestValidateDescription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		val      string
		wantCode string
	}{
		{name: "valid", val: "Description"},
		{name: "at min length", val: strings.Repeat("a", 1)},
		{name: "at max length", val: strings.Repeat("\u00E9", DescriptionMaxLength)},
		{name: "normalized", val: "Caf\u00E9"},
		{name: "untrimmed", val: " Description ", wantCode: validator.CodeUntrimmed},
		{name: "not normalized", val: "Cafe\u0301", wantCode: validator.CodeNormalized},
		{name: "empty", val: "", wantCode: validator.CodeRequired},
		{name: "over max length", val: strings.Repeat("\u00E9", DescriptionMaxLength+1), wantCode: validator.CodeMaxLength},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateDescription(tt.val)
			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			fe, ok := errors.AsType[*validator.FieldError](err)
			if !ok {
				t.Fatalf("error: got %v, want *validator.FieldError", err)
			}
			if fe.Field != "description" || fe.Code != tt.wantCode {
				t.Errorf("field error: got %s/%s, want description/%s", fe.Field, fe.Code, tt.wantCode)
			}
		})
	}
}
