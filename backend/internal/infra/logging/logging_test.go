//go:build unit

package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func newJSONLogger(t *testing.T, level slog.Leveler) (*slog.Logger, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	log, err := New(Options{Level: level, Format: FormatJSON, Output: &buf})
	if err != nil {
		t.Fatal(err)
	}
	return log, &buf
}

// records decodes one JSON object per line of buf.
func records(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for line := range strings.Lines(buf.String()) {
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("unmarshal %q: %v", line, err)
		}
		out = append(out, rec)
	}
	return out
}

func oneRecord(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	recs := records(t, buf)
	if len(recs) != 1 {
		t.Fatalf("records: got %d, want 1\n%s", len(recs), buf.String())
	}
	return recs[0]
}

func TestWithAttrs(t *testing.T) {
	t.Parallel()

	t.Run("attrs from ctx appear on records", func(t *testing.T) {
		t.Parallel()
		log, buf := newJSONLogger(t, nil)

		ctx := WithAttrs(context.Background(), slog.String("request_id", "abc"))
		log.InfoContext(ctx, "msg", "k", "v")

		rec := oneRecord(t, buf)
		if rec["request_id"] != "abc" {
			t.Errorf("request_id: got %v, want %q", rec["request_id"], "abc")
		}
		if rec["k"] != "v" {
			t.Errorf("k: got %v, want %q", rec["k"], "v")
		}
	})

	t.Run("no attrs in ctx", func(t *testing.T) {
		t.Parallel()
		log, buf := newJSONLogger(t, nil)

		log.InfoContext(context.Background(), "msg")

		rec := oneRecord(t, buf)
		if _, ok := rec["request_id"]; ok {
			t.Errorf("unexpected request_id: %v", rec["request_id"])
		}
	})

	t.Run("nested calls accumulate", func(t *testing.T) {
		t.Parallel()
		log, buf := newJSONLogger(t, nil)

		ctx := WithAttrs(context.Background(), slog.String("a", "1"))
		ctx = WithAttrs(ctx, slog.String("b", "2"))
		log.InfoContext(ctx, "msg")

		rec := oneRecord(t, buf)
		if rec["a"] != "1" || rec["b"] != "2" {
			t.Errorf("attrs: got a=%v b=%v, want a=1 b=2", rec["a"], rec["b"])
		}
	})

	t.Run("sibling contexts do not alias", func(t *testing.T) {
		t.Parallel()
		log, buf := newJSONLogger(t, nil)

		// Three appends leave the parent's slice with len 3 and cap 4, so an
		// unclipped append would write both siblings into the same slot.
		parent := WithAttrs(context.Background(), slog.String("parent", "p"))
		parent = WithAttrs(parent, slog.String("parent2", "p2"))
		parent = WithAttrs(parent, slog.String("parent3", "p3"))

		left := WithAttrs(parent, slog.String("side", "left"))
		right := WithAttrs(parent, slog.String("side", "right"))

		log.InfoContext(left, "left")
		log.InfoContext(right, "right")
		log.InfoContext(parent, "parent")

		recs := records(t, buf)
		if len(recs) != 3 {
			t.Fatalf("records: got %d, want 3\n%s", len(recs), buf.String())
		}
		if recs[0]["side"] != "left" {
			t.Errorf("left side: got %v, want %q", recs[0]["side"], "left")
		}
		if recs[1]["side"] != "right" {
			t.Errorf("right side: got %v, want %q", recs[1]["side"], "right")
		}
		if _, ok := recs[2]["side"]; ok {
			t.Errorf("parent side: got %v, want absent", recs[2]["side"])
		}
		for i, rec := range recs {
			if rec["parent"] != "p" || rec["parent2"] != "p2" || rec["parent3"] != "p3" {
				t.Errorf("record %d parent attrs: got %v %v %v", i, rec["parent"], rec["parent2"], rec["parent3"])
			}
		}
	})
}

func TestHandler_WithAttrsWithGroup(t *testing.T) {
	t.Parallel()

	t.Run("logger With keeps ctx attrs", func(t *testing.T) {
		t.Parallel()
		log, buf := newJSONLogger(t, nil)

		ctx := WithAttrs(context.Background(), slog.String("request_id", "abc"))
		log.With("component", "x").InfoContext(ctx, "msg")

		rec := oneRecord(t, buf)
		if rec["component"] != "x" {
			t.Errorf("component: got %v, want %q", rec["component"], "x")
		}
		if rec["request_id"] != "abc" {
			t.Errorf("request_id: got %v, want %q", rec["request_id"], "abc")
		}
	})

	t.Run("logger WithGroup keeps ctx attrs", func(t *testing.T) {
		t.Parallel()
		log, buf := newJSONLogger(t, nil)

		ctx := WithAttrs(context.Background(), slog.String("request_id", "abc"))
		log.WithGroup("g").InfoContext(ctx, "msg", "k", "v")

		rec := oneRecord(t, buf)
		g, ok := rec["g"].(map[string]any)
		if !ok {
			t.Fatalf("group g: got %v, want object", rec["g"])
		}
		if g["k"] != "v" {
			t.Errorf("g.k: got %v, want %q", g["k"], "v")
		}
		// Record attrs added in Handle land inside the open group.
		if g["request_id"] != "abc" {
			t.Errorf("g.request_id: got %v, want %q", g["request_id"], "abc")
		}
	})

	t.Run("handlers stay wrapped", func(t *testing.T) {
		t.Parallel()
		log, _ := newJSONLogger(t, nil)

		if _, ok := log.With("k", "v").Handler().(ctxHandler); !ok {
			t.Errorf("With: got %T, want ctxHandler", log.With("k", "v").Handler())
		}
		if _, ok := log.WithGroup("g").Handler().(ctxHandler); !ok {
			t.Errorf("WithGroup: got %T, want ctxHandler", log.WithGroup("g").Handler())
		}
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		format   Format
		wantJSON bool
	}{
		{name: "json", format: FormatJSON, wantJSON: true},
		{name: "text", format: FormatText, wantJSON: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			log, err := New(Options{Format: tt.format, Output: &buf})
			if err != nil {
				t.Fatal(err)
			}
			ctx := WithAttrs(context.Background(), slog.String("request_id", "abc"))
			log.InfoContext(ctx, "hello")

			out := buf.String()
			if got := json.Valid([]byte(out)); got != tt.wantJSON {
				t.Errorf("json: got %v, want %v\n%s", got, tt.wantJSON, out)
			}
			if !tt.wantJSON && !strings.Contains(out, "msg=hello") {
				t.Errorf("text output missing msg=hello: %s", out)
			}
			if !strings.Contains(out, "abc") {
				t.Errorf("output missing ctx attr: %s", out)
			}
		})
	}
}

func TestNew_UnknownFormat(t *testing.T) {
	t.Parallel()

	for _, f := range []Format{0, FormatText + 1} {
		t.Run(f.String(), func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			log, err := New(Options{Format: f, Output: &buf})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if log != nil {
				t.Errorf("logger: got %v, want nil", log)
			}
		})
	}
}

func TestNew_Level(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		level     slog.Leveler
		wantDebug bool
		wantInfo  bool
	}{
		{name: "nil defaults to info", level: nil, wantDebug: false, wantInfo: true},
		{name: "debug", level: slog.LevelDebug, wantDebug: true, wantInfo: true},
		{name: "warn", level: slog.LevelWarn, wantDebug: false, wantInfo: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			log, buf := newJSONLogger(t, tt.level)

			log.Debug("debug")
			log.Info("info")
			log.Warn("warn")

			out := buf.String()
			if got := strings.Contains(out, `"msg":"debug"`); got != tt.wantDebug {
				t.Errorf("debug logged: got %v, want %v", got, tt.wantDebug)
			}
			if got := strings.Contains(out, `"msg":"info"`); got != tt.wantInfo {
				t.Errorf("info logged: got %v, want %v", got, tt.wantInfo)
			}
			if !strings.Contains(out, `"msg":"warn"`) {
				t.Errorf("warn not logged: %s", out)
			}
		})
	}

	t.Run("level var changes apply", func(t *testing.T) {
		t.Parallel()
		var lv slog.LevelVar
		lv.Set(slog.LevelError)
		log, buf := newJSONLogger(t, &lv)

		log.Warn("before")
		lv.Set(slog.LevelWarn)
		log.Warn("after")

		rec := oneRecord(t, buf)
		if rec["msg"] != "after" {
			t.Errorf("msg: got %v, want %q", rec["msg"], "after")
		}
	})
}
