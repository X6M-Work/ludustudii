// Package domain holds the rules and constants shared by the learning card
// endpoints.
package domain

// CardsPath is the base URL path of the learning cards resource.
const CardsPath = "/api/lcards"

// Page size limits for listing learning cards. ListDefaultLimit applies when
// the request gives no limit.
const (
	ListDefaultLimit = 20
	ListMinLimit     = 1
	ListMaxLimit     = 100
)
