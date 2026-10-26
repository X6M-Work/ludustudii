package validator

import "errors"

var (
	ErrInvalidUUID7 = errors.New("is invalid uuid v7")
	ErrRequired     = errors.New("is required")
	ErrMaxLength    = errors.New("exceeds maximum length")
)

type FieldError struct {
	Field string `json:"field"`
	Code  string `json:"code"`
	Err   error  `json:"-"`
}

func NewFieldError(field, code string, err error) *FieldError {
	return &FieldError{
		Field: field,
		Code:  code,
		Err:   err,
	}
}

func (ve *FieldError) Error() string {
	return ve.Err.Error()
}
