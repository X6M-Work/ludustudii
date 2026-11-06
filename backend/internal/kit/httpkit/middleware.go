package httpkit

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"
)

// Middleware wraps a handler with request or response behavior.
type Middleware func(http.Handler) http.Handler

// Chain combines middleware in request order: the first runs outermost.
// With no middleware, the returned wrapper leaves the handler unchanged.
func Chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for _, middleware := range slices.Backward(middlewares) {
			next = middleware(next)
		}
		return next
	}
}

// Recover converts panics to central errors, while allowing ErrAbortHandler
// to terminate a connection as required by net/http.
func Recover(onError ErrorHandler) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if p := recover(); p != nil {
					if p == http.ErrAbortHandler { //nolint:errorlint // net/http rethrows only the exact abort sentinel.
						panic(p)
					}
					err, ok := p.(error)
					if !ok {
						err = fmt.Errorf("%v", p)
					}
					onError(w, r, err)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestID reuses the first incoming X-Request-Id when it contains 1–128
// ASCII letters, digits, dots, underscores or hyphens. Otherwise it generates a
// random base32 string. It sets the response header and passes next a
// request whose context is enriched. Because next receives a new request,
// RequestID must run outside RequestLogger, which reads the routed request.
func RequestID(withID func(context.Context, string) context.Context) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-Id")
			if !validRequestID(id) {
				id = rand.Text()
			}
			w.Header().Set("X-Request-Id", id)
			next.ServeHTTP(w, r.WithContext(withID(r.Context(), id)))
		})
	}
}

const requestIDMaxLength = 128

func validRequestID(id string) bool {
	if len(id) == 0 || len(id) > requestIDMaxLength {
		return false
	}
	for i := range len(id) {
		ch := id[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '.' || ch == '_' || ch == '-' {
			continue
		}
		return false
	}
	return true
}

// RequestLoggerOptions configures request summary logging.
type RequestLoggerOptions struct {
	// QuietPaths is a set of exact URL paths whose summaries use debug at
	// every status. All other paths keep the normal status-based levels.
	QuietPaths map[string]struct{}
}

// RequestLogger logs one request record after the final response, including
// errors and panics handled by next. The route is taken from Request.Pattern,
// which http.ServeMux sets on the request it receives, so middleware between
// RequestLogger and the mux must pass the same request on.
func RequestLogger(log *slog.Logger, options RequestLoggerOptions) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			status := http.StatusOK
			if tracked, ok := w.(*ResponseWriter); ok {
				status = tracked.Status
			}
			_, quiet := options.QuietPaths[r.URL.Path]
			level := slog.LevelInfo
			switch {
			case quiet:
				level = slog.LevelDebug
			case status >= http.StatusInternalServerError:
				level = slog.LevelError
			case status >= http.StatusBadRequest:
				level = slog.LevelWarn
			}
			log.LogAttrs(r.Context(), level, "request",
				slog.String("method", r.Method), slog.String("route", r.Pattern),
				slog.String("path", r.URL.Path), slog.Int("status", status),
				slog.Duration("latency", time.Since(start)))
		})
	}
}

// BodyLimitOptions configures request body limits.
type BodyLimitOptions struct {
	// ExcludedPaths is a set of exact URL paths that bypass both the declared
	// length check and the reader limit. A nil set excludes no paths.
	ExcludedPaths map[string]struct{}
}

// BodyLimit rejects known oversized bodies before dispatch and limits bytes
// read with http.MaxBytesReader, except for paths excluded by options.
func BodyLimit(limit int64, onError ErrorHandler, options BodyLimitOptions) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, excluded := options.ExcludedPaths[r.URL.Path]; excluded {
				next.ServeHTTP(w, r)
				return
			}
			if r.ContentLength > limit {
				onError(w, r, ErrPayloadTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}
