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
		{name: "single character", val: "a"},
		{name: "at max length single-byte", val: strings.Repeat("a", TitleMaxLength)},
		{name: "at max length multi-byte", val: strings.Repeat("é", TitleMaxLength)},
		{name: "empty", val: "", wantCode: validator.CodeRequired},
		{name: "whitespace", val: " \t\n ", wantCode: validator.CodeRequired},
		{name: "over max length single-byte", val: strings.Repeat("a", TitleMaxLength+1), wantCode: validator.CodeMaxLength},
		{name: "over max length multi-byte", val: strings.Repeat("é", TitleMaxLength+1), wantCode: validator.CodeMaxLength},
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
		{name: "single character", val: "a"},
		{name: "at max length single-byte", val: strings.Repeat("a", DescriptionMaxLength)},
		{name: "at max length multi-byte", val: strings.Repeat("é", DescriptionMaxLength)},
		{name: "empty", val: "", wantCode: validator.CodeRequired},
		{name: "whitespace", val: " \t\n ", wantCode: validator.CodeRequired},
		{name: "over max length single-byte", val: strings.Repeat("a", DescriptionMaxLength+1), wantCode: validator.CodeMaxLength},
		{name: "over max length multi-byte", val: strings.Repeat("é", DescriptionMaxLength+1), wantCode: validator.CodeMaxLength},
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
