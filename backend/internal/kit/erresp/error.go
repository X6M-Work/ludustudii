// Package erresp defines the JSON body of API error responses.
package erresp

import (
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
)

// Machine-readable error codes sent in ErrorResponse.Code.
const (
	CodeBinding              = "bind_error"
	CodeMethodNotAllowed     = "method_not_allowed"
	CodeNotFound             = "not_found"
	CodePayloadTooLarge      = "payload_too_large"
	CodeUnexpected           = "unexpected_error"
	CodeValidation           = "validation_error"
	CodeParam                = "param_error"
	CodeUnsupportedMediaType = "unsupported_media_type"
)

// ErrorResponse is the JSON body of every API error response. Errors lists
// the per-field failures of a validation error and is omitted otherwise.
type ErrorResponse struct {
	Code   string                  `json:"code"`
	Errors []*validator.FieldError `json:"errors,omitempty"`
}

// New returns an ErrorResponse with the given code. Prefer the named
// constructors for the codes this package defines.
func New(code string) ErrorResponse {
	return ErrorResponse{
		Code: code,
	}
}

// Binding returns the response for a request body that could not be decoded.
func Binding() ErrorResponse {
	return ErrorResponse{
		Code: CodeBinding,
	}
}

// MethodNotAllowed returns the response for a method the route does not support.
func MethodNotAllowed() ErrorResponse {
	return ErrorResponse{
		Code: CodeMethodNotAllowed,
	}
}

// NotFound returns the response for a route or resource that does not exist.
func NotFound() ErrorResponse {
	return ErrorResponse{
		Code: CodeNotFound,
	}
}

// PayloadTooLarge returns the response for a request body over the size limit.
func PayloadTooLarge() ErrorResponse {
	return ErrorResponse{
		Code: CodePayloadTooLarge,
	}
}

// UnsupportedMediaType returns the response for a request body whose
// Content-Type the endpoint does not accept.
func UnsupportedMediaType() ErrorResponse {
	return ErrorResponse{
		Code: CodeUnsupportedMediaType,
	}
}

// Unexpected returns the response for a server-side failure; it carries no details.
func Unexpected() ErrorResponse {
	return ErrorResponse{
		Code: CodeUnexpected,
	}
}

// Validation returns a validation error response listing every
// *validator.FieldError found in err's tree, in depth-first order. Errors of
// other types are skipped.
func Validation(err error) ErrorResponse {
	var errs []*validator.FieldError

	var collect func(error)
	collect = func(err error) {
		if fe, ok := err.(*validator.FieldError); ok {
			errs = append(errs, fe)
			return
		}
		switch w := err.(type) {
		case interface{ Unwrap() []error }:
			for _, e := range w.Unwrap() {
				collect(e)
			}
		case interface{ Unwrap() error }:
			collect(w.Unwrap())
		}
	}
	collect(err)

	return ErrorResponse{
		Code:   CodeValidation,
		Errors: errs,
	}
}

// Param returns the response for an invalid path parameter.
func Param() ErrorResponse {
	return ErrorResponse{
		Code: CodeParam,
	}
}
