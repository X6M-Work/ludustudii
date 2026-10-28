package erresp

import (
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
)

const (
	CodeBinding          = "bind_error"
	CodeMethodNotAllowed = "method_not_allowed"
	CodeNotFound         = "not_found"
	CodePayloadTooLarge  = "payload_too_large"
	CodeUnexpected       = "unexpected_error"
	CodeValidation       = "validation_error"
	CodeParam            = "param_error"
)

type ErrorResponse struct {
	Code   string                  `json:"code"`
	Errors []*validator.FieldError `json:"errors,omitempty"`
}

func New(code string) ErrorResponse {
	return ErrorResponse{
		Code: code,
	}
}

func Binding() ErrorResponse {
	return ErrorResponse{
		Code: CodeBinding,
	}
}

func MethodNotAllowed() ErrorResponse {
	return ErrorResponse{
		Code: CodeMethodNotAllowed,
	}
}

func NotFound() ErrorResponse {
	return ErrorResponse{
		Code: CodeNotFound,
	}
}

func PayloadTooLarge() ErrorResponse {
	return ErrorResponse{
		Code: CodePayloadTooLarge,
	}
}

func Unexpected() ErrorResponse {
	return ErrorResponse{
		Code: CodeUnexpected,
	}
}

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

func Param() ErrorResponse {
	return ErrorResponse{
		Code: CodeParam,
	}
}
