package domain

import "errors"

// CardsPath is the base URL path of the lcards resource.
const CardsPath = "/api/lcards"

// Log messages for learning cards domain errors.
const (
	TextInvalidID  = "The parameter 'id' is invalid"
	TextBinding    = "Failed to bind the request body"
	TextValidation = "The request validation failed"
	TextNotFound   = "The learning card was not found"
	TextUnexpected = "An unexpected error occurred"
)

// List pagination limits for learning cards.
const (
	ListDefaultLimit = 20
	ListMinLimit     = 1
	ListMaxLimit     = 100
)

// Domain errors for learning cards.
var (
	ErrNotFound = errors.New("not found")
)
