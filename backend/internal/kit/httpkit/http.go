// Package httpkit provides error-returning HTTP handlers, responses and middleware.
package httpkit

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
)

// HandlerFunc handles a request and returns errors for the central error handler.
type HandlerFunc func(http.ResponseWriter, *http.Request) error

// ErrorHandler reports an unhandled request error.
type ErrorHandler func(http.ResponseWriter, *http.Request, error)

// Adapt routes returned errors to onError.
func Adapt(h HandlerFunc, onError ErrorHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			onError(w, r, err)
		}
	})
}

// StatusError carries a public HTTP status and response body, plus an optional
// underlying cause that is logged without exposing it to clients.
type StatusError struct {
	Code     int
	Err      error
	Response erresp.ErrorResponse
}

func (e *StatusError) Error() string {
	if e.Err != nil {
		return http.StatusText(e.Code) + ": " + e.Err.Error()
	}
	return http.StatusText(e.Code)
}

// Unwrap returns the underlying error.
func (e *StatusError) Unwrap() error { return e.Err }

// Shared errors carry their HTTP statuses and response bodies.
var (
	ErrNotFound             = &StatusError{Code: http.StatusNotFound, Response: erresp.NotFound()}
	ErrMethodNotAllowed     = &StatusError{Code: http.StatusMethodNotAllowed, Response: erresp.MethodNotAllowed()}
	ErrPayloadTooLarge      = &StatusError{Code: http.StatusRequestEntityTooLarge, Response: erresp.PayloadTooLarge()}
	ErrUnsupportedMediaType = &StatusError{Code: http.StatusUnsupportedMediaType, Response: erresp.UnsupportedMediaType()}
	ErrPreconditionFailed   = &StatusError{Code: http.StatusPreconditionFailed, Response: erresp.PreconditionFailed()}
	ErrUnexpected           = &StatusError{Code: http.StatusInternalServerError, Response: erresp.Unexpected()}
)

// StatusCode returns the status carried by err, or zero for an untyped error.
func StatusCode(err error) int {
	if status, ok := errors.AsType[*StatusError](err); ok {
		return status.Code
	}
	return 0
}

// JSON writes a JSON response with a trailing newline.
func JSON(w http.ResponseWriter, status int, value any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(value)
}

// NoContent writes status without a response body.
func NoContent(w http.ResponseWriter, status int) error {
	w.WriteHeader(status)
	return nil
}

// Text writes a UTF-8 plain-text response.
func Text(w http.ResponseWriter, status int, value string) error {
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	w.WriteHeader(status)
	_, err := io.WriteString(w, value)
	return err
}

// BindJSON binds only the JSON body, ignoring query and path values. A known
// empty body leaves target unchanged; other bodies require application/json
// (parameters ignored). Malformed JSON returns 400 and oversized input 413.
func BindJSON(r *http.Request, target any) error {
	if r.ContentLength == 0 {
		return nil
	}
	base, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";")
	if strings.TrimSpace(base) != "application/json" {
		return ErrUnsupportedMediaType
	}
	data, err := io.ReadAll(r.Body)
	if err == nil {
		err = json.Unmarshal(data, target)
	}
	if err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return ErrPayloadTooLarge
		}
		if _, ok := errors.AsType[*StatusError](err); ok {
			return err
		}
		return &StatusError{Code: http.StatusBadRequest, Err: err, Response: erresp.Binding()}
	}
	return nil
}
