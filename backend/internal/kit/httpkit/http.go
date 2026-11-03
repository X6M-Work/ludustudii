// Package httpkit provides error-returning HTTP handlers, responses and middleware.
package httpkit

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
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

// StatusError carries a public HTTP status and an optional underlying error.
type StatusError struct {
	Code int
	Err  error
}

func (e *StatusError) Error() string { return http.StatusText(e.Code) }

// Unwrap returns the underlying error.
func (e *StatusError) Unwrap() error { return e.Err }

// Routing and binding errors retain their named HTTP statuses.
var (
	ErrNotFound             = &StatusError{Code: http.StatusNotFound}
	ErrMethodNotAllowed     = &StatusError{Code: http.StatusMethodNotAllowed}
	ErrPayloadTooLarge      = &StatusError{Code: http.StatusRequestEntityTooLarge}
	ErrUnsupportedMediaType = &StatusError{Code: http.StatusUnsupportedMediaType}
)

// StatusCode returns the status carried by err, or zero for an untyped error.
// MaxBytesReader errors carry the payload-too-large status.
func StatusCode(err error) int {
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return http.StatusRequestEntityTooLarge
	}
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
func BindJSON(_ http.ResponseWriter, r *http.Request, target any) error {
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
		if StatusCode(err) != 0 {
			return err
		}
		return &StatusError{Code: http.StatusBadRequest, Err: err}
	}
	return nil
}
