package validator

import "errors"

// Sentinel errors wrapped by FieldError, one per check.
var (
	ErrInvalidUUID7   = errors.New("is invalid uuid v7")
	ErrUntrimmed      = errors.New("contains leading or trailing whitespace")
	ErrNormalized     = errors.New("is not normalized")
	ErrRequired       = errors.New("is required")
	ErrMaxLength      = errors.New("exceeds maximum length")
	ErrInvalidInteger = errors.New("is invalid integer")
	ErrOutOfRange     = errors.New("is out of range")
)

// FieldError reports a failed check on one input field. Field and Code are
// sent to the client; Err is for logs and errors.Is.
type FieldError struct {
	Field string `json:"field"`
	Code  string `json:"code"`
	Err   error  `json:"-"`
}

// NewFieldError returns a FieldError for field with the given code and cause.
func NewFieldError(field, code string, err error) *FieldError {
	return &FieldError{
		Field: field,
		Code:  code,
		Err:   err,
	}
}

// Error returns the message of the wrapped error.
func (fe *FieldError) Error() string {
	return fe.Err.Error()
}

// Unwrap exposes the underlying sentinel (ErrRequired, ErrMaxLength, ...) so
// errors.Is works on a *FieldError and on joined errors that contain one.
func (fe *FieldError) Unwrap() error { return fe.Err }
