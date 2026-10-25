package domain

import (
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
)

const (
	TitleMaxLength       = 200
	DescriptionMaxLength = 2000
)

func ValidateTitle(val string) error {
	if err := validator.Required("title", val); err != nil {
		return err
	}
	if err := validator.MaxLength("title", val, TitleMaxLength); err != nil {
		return err
	}
	return nil
}

func ValidateDescription(val string) error {
	if err := validator.Required("description", val); err != nil {
		return err
	}
	if err := validator.MaxLength("description", val, DescriptionMaxLength); err != nil {
		return err
	}
	return nil
}
