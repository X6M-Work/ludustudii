package domain

import (
	"strings"
	"unicode/utf8"
)

const (
	MaxTitleLength       = 200
	MaxDescriptionLength = 2000
)

type Card struct {
	ID          string
	Title       string
	Description string
}

func CreateCard(title, description string) (Card, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)

	if err := validateCardContent(title, description); err != nil {
		return Card{}, err
	}

	return Card{
		Title:       title,
		Description: description,
	}, nil
}

func UpdateCard(id, title, description string) (Card, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)

	if err := validateCardContent(title, description); err != nil {
		return Card{}, err
	}

	return Card{
		ID:          id,
		Title:       title,
		Description: description,
	}, nil
}

func validateCardContent(title, description string) error {
	if title == "" {
		return ErrTitleRequired
	}
	if utf8.RuneCountInString(title) > MaxTitleLength {
		return ErrTitleTooLong
	}
	if description == "" {
		return ErrDescriptionRequired
	}
	if utf8.RuneCountInString(description) > MaxDescriptionLength {
		return ErrDescriptionTooLong
	}
	return nil
}

