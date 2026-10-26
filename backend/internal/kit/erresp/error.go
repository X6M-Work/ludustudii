package erresp

import (
	"errors"

	"github.com/apotourlyan/ludustudii/internal/kit/validator"
)

const (
	CodeBinding    = "bind_error"
	CodeNotFound   = "not_found"
	CodeUnexpected = "unexpected_error"
	CodeValidation = "validation_error"
	CodeParam      = "param_error"
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

func NotFound() ErrorResponse {
	return ErrorResponse{
		Code: CodeNotFound,
	}
}

func Unexpected() ErrorResponse {
	return ErrorResponse{
		Code: CodeUnexpected,
	}
}

func Validation(err error) ErrorResponse {
	var errs []*validator.FieldError

	if w, ok := err.(interface{ Unwrap() []error }); ok {
		for _, e := range w.Unwrap() {
			fe, ok := errors.AsType[*validator.FieldError](e)
			if ok {
				errs = append(errs, fe)
			}
		}
	} else if fe, ok := errors.AsType[*validator.FieldError](err); ok {
		errs = append(errs, fe)
	}

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
