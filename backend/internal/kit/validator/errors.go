package validator

import "errors"

var (
	ErrInvalidUUID7 = errors.New("is invalid uuid v7")
	ErrRequired     = errors.New("is required")
	ErrMaxLength    = errors.New("exceeds maximum length")
)
