package httpkit

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
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

// RequestID reuses the first incoming X-Request-Id, or generates 32 ASCII
// letters when it is empty. It sets the response header and passes next a
// request whose context is enriched. Because next receives a new request,
// RequestID must run outside RequestLogger, which reads the routed request.
func RequestID(withID func(context.Context, string) context.Context) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-Id")
			if id == "" {
				id = requestID()
			}
			w.Header().Set("X-Request-Id", id)
			next.ServeHTTP(w, r.WithContext(withID(r.Context(), id)))
		})
	}
}

func requestID() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var id [32]byte
	for i := range id {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			panic(err)
		}
		id[i] = letters[n.Int64()]
	}
	return string(id[:])
}

// RequestLogger logs one request record after the final response, including
// errors and panics handled by next. The route is taken from Request.Pattern,
// which http.ServeMux sets on the request it receives, so middleware between
// RequestLogger and the mux must pass the same request on.
func RequestLogger(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			status := http.StatusOK
			if tracked, ok := w.(*ResponseWriter); ok {
				status = tracked.Status
			}
			level := slog.LevelInfo
			switch {
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

// BodyLimit rejects known oversized bodies before dispatch and limits bytes
// read from unknown-length bodies with http.MaxBytesReader.
func BodyLimit(limit int64, onError ErrorHandler) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > limit {
				onError(w, r, ErrPayloadTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}
