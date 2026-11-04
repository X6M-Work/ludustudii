package httpkit

import (
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
)

// Param reports an invalid path parameter, retaining its cause for logging.
func Param(err error) error {
	return &StatusError{Code: http.StatusBadRequest, Err: err, Response: erresp.Param()}
}

// Validation reports field validation failures, retaining their causes.
func Validation(err error) error {
	return &StatusError{Code: http.StatusBadRequest, Err: err, Response: erresp.Validation(err)}
}

// PreconditionRequired reports a missing or invalid If-Match header.
func PreconditionRequired(err error) error {
	return &StatusError{Code: http.StatusPreconditionRequired, Err: err, Response: erresp.PreconditionRequired()}
}
