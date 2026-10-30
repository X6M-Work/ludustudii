//go:build unit

package env

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/infra/logging"
)

// setRequired sets the vars Load needs and clears the logging vars so the
// host environment cannot leak into a test.
func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv(EnvDBConnectionString, "postgres://user:secret@localhost:5432/db")
	t.Setenv(EnvAppPort, "8080")
	t.Setenv(EnvLogLevel, "")
	t.Setenv(EnvLogFormat, "")
}

func TestLoad_LogDefaults(t *testing.T) {
	setRequired(t)

	v, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := v.LogLevel.Level(); got != slog.LevelInfo {
		t.Errorf("level: got %v, want %v", got, slog.LevelInfo)
	}
	if v.LogFormat != logging.FormatJSON {
		t.Errorf("format: got %v, want %v", v.LogFormat, logging.FormatJSON)
	}
}

func TestLoad_LogLevel(t *testing.T) {
	tests := []struct {
		val  string
		want slog.Level
	}{
		{val: "debug", want: slog.LevelDebug},
		{val: "DEBUG", want: slog.LevelDebug},
		{val: "Info", want: slog.LevelInfo},
		{val: "warn", want: slog.LevelWarn},
		{val: "error", want: slog.LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			setRequired(t)
			t.Setenv(EnvLogLevel, tt.val)

			v, err := Load()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := v.LogLevel.Level(); got != tt.want {
				t.Errorf("level: got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoad_LogFormat(t *testing.T) {
	tests := []struct {
		val  string
		want logging.Format
	}{
		{val: "json", want: logging.FormatJSON},
		{val: "text", want: logging.FormatText},
	}

	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			setRequired(t)
			t.Setenv(EnvLogFormat, tt.val)

			v, err := Load()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v.LogFormat != tt.want {
				t.Errorf("format: got %v, want %v", v.LogFormat, tt.want)
			}
		})
	}
}

func TestLoad_LogErrors(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		format   string
		wantErrs []string
	}{
		{name: "invalid level", level: "verbose", wantErrs: []string{EnvLogLevel}},
		{name: "invalid format", format: "xml", wantErrs: []string{EnvLogFormat}},
		{name: "format is case-sensitive", format: "JSON", wantErrs: []string{EnvLogFormat}},
		{name: "both collected", level: "loud", format: "yaml", wantErrs: []string{EnvLogLevel, EnvLogFormat}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRequired(t)
			t.Setenv(EnvLogLevel, tt.level)
			t.Setenv(EnvLogFormat, tt.format)

			v, err := Load()
			if err == nil {
				t.Fatalf("expected error, got vars %+v", v)
			}
			for _, want := range tt.wantErrs {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not mention %s", err, want)
				}
			}
		})
	}
}

func TestVars_LogAttrs(t *testing.T) {
	setRequired(t)
	t.Setenv(EnvLogLevel, "warn")
	t.Setenv(EnvLogFormat, "text")

	v, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := map[string]string{}
	for _, a := range v.LogAttrs() {
		got[a.Key] = a.Value.String()
	}
	if strings.Contains(got[EnvDBConnectionString], "secret") {
		t.Errorf("%s not redacted: %q", EnvDBConnectionString, got[EnvDBConnectionString])
	}
	if got[EnvLogLevel] != "WARN" {
		t.Errorf("%s: got %q, want %q", EnvLogLevel, got[EnvLogLevel], "WARN")
	}
	if got[EnvLogFormat] != "text" {
		t.Errorf("%s: got %q, want %q", EnvLogFormat, got[EnvLogFormat], "text")
	}
}
