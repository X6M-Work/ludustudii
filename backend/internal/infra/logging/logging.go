// Package logging builds slog loggers that add request-scoped attrs stored in
// the context.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
)

type ctxKey struct{}

// WithAttrs returns a copy of ctx that carries attrs in addition to any
// already stored in it. Loggers built by New add them to every record logged
// with that context.
func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	prev, _ := ctx.Value(ctxKey{}).([]slog.Attr)
	// Clip so that sibling contexts derived from the same parent never append
	// into a shared backing array.
	return context.WithValue(ctx, ctxKey{}, append(slices.Clip(prev), attrs...))
}

// ctxHandler adds the attrs stored in the record's context before delegating.
type ctxHandler struct {
	inner slog.Handler
}

func (h ctxHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h ctxHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs, ok := ctx.Value(ctxKey{}).([]slog.Attr); ok {
		r.AddAttrs(attrs...)
	}
	return h.inner.Handle(ctx, r)
}

func (h ctxHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return ctxHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h ctxHandler) WithGroup(name string) slog.Handler {
	return ctxHandler{inner: h.inner.WithGroup(name)}
}

// Options configures a logger built by New.
type Options struct {
	Level  slog.Leveler
	Format Format
	Output io.Writer
}

// New builds a logger that writes opts.Format records at opts.Level or above
// to opts.Output and includes the attrs stored in the context by WithAttrs. A
// nil opts.Level means slog.LevelInfo. It returns an error if opts.Format is
// not FormatJSON or FormatText.
func New(opts Options) (*slog.Logger, error) {
	ho := &slog.HandlerOptions{Level: opts.Level}
	var h slog.Handler
	switch opts.Format {
	case FormatJSON:
		h = slog.NewJSONHandler(opts.Output, ho)
	case FormatText:
		h = slog.NewTextHandler(opts.Output, ho)
	default:
		return nil, fmt.Errorf("unknown format %v", opts.Format)
	}
	return slog.New(ctxHandler{inner: h}), nil
}
