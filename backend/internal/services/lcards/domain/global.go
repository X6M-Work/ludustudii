// Package domain holds the rules and constants shared by the learning card
// endpoints.
package domain

import "errors"

// CardsPath is the base URL path of the learning cards resource.
const CardsPath = "/api/lcards"

// Log messages the learning card handlers emit when a request fails.
const (
	TextInvalidID  = "The parameter 'id' is invalid"
	TextBinding    = "Failed to bind the request body"
	TextValidation = "The request validation failed"
	TextNotFound   = "The learning card was not found"
	TextUnexpected = "An unexpected error occurred"
)

// Page size limits for listing learning cards. ListDefaultLimit applies when
// the request gives no limit.
const (
	ListDefaultLimit = 20
	ListMinLimit     = 1
	ListMaxLimit     = 100
)

// Domain errors for learning cards. Stores return ErrNotFound when no card
// has the requested ID.
var (
	ErrNotFound = errors.New("not found")
)
