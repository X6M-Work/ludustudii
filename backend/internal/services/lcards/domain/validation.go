package domain

import (
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
)

// Maximum lengths of card fields, in runes.
const (
	TitleMaxLength       = 200
	DescriptionMaxLength = 2000
)

// ValidateTitle checks that a card title is trimmed, in NFC, not empty and at
// most TitleMaxLength runes. It returns the first failed check as a
// *validator.FieldError for the "title" field.
func ValidateTitle(val string) error {
	if err := validator.Trimmed("title", val); err != nil {
		return err
	}
	if err := validator.Normalized("title", val); err != nil {
		return err
	}
	if err := validator.Required("title", val); err != nil {
		return err
	}
	if err := validator.MaxLength("title", val, TitleMaxLength); err != nil {
		return err
	}
	return nil
}

// ValidateDescription checks that a card description is trimmed, in NFC, not
// empty and at most DescriptionMaxLength runes. It returns the first failed
// check as a *validator.FieldError for the "description" field.
func ValidateDescription(val string) error {
	if err := validator.Trimmed("description", val); err != nil {
		return err
	}
	if err := validator.Normalized("description", val); err != nil {
		return err
	}
	if err := validator.Required("description", val); err != nil {
		return err
	}
	if err := validator.MaxLength("description", val, DescriptionMaxLength); err != nil {
		return err
	}
	return nil
}
