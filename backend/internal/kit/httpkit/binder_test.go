//go:build unit

package httpkit

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
)

func TestJSONBinder(t *testing.T) {
	t.Parallel()

	type payload struct {
		Title string `json:"title"`
	}

	tests := []struct {
		name        string
		contentType string
		body        string
		want        payload
		wantErr     error
		wantStatus  int
	}{
		{
			name:        "json / valid",
			contentType: "application/json",
			body:        `{"title":"Hello"}`,
			want:        payload{Title: "Hello"},
		},
		{
			name:        "json with charset / valid",
			contentType: "application/json; charset=utf-8",
			body:        `{"title":"Hello"}`,
			want:        payload{Title: "Hello"},
		},
		{
			name:        "json / empty / nil",
			contentType: "application/json",
			body:        "",
			want:        payload{},
		},
		{
			name:        "none / empty / nil",
			contentType: "",
			body:        "",
			want:        payload{},
		},
		{
			name:        "none / non-empty / unsupported media type",
			contentType: "",
			body:        `{"title":"Hello"}`,
			wantErr:     ErrUnsupportedMediaType,
			wantStatus:  http.StatusUnsupportedMediaType,
		},
		{
			name:        "xml / non-empty / unsupported media type",
			contentType: "application/xml",
			body:        `<payload><Title>Hello</Title></payload>`,
			wantErr:     ErrUnsupportedMediaType,
			wantStatus:  http.StatusUnsupportedMediaType,
		},
		{
			name:        "json / malformed / bad request",
			contentType: "application/json",
			body:        `{"title":`,
			wantStatus:  http.StatusBadRequest,
		},
		{name: "trailing JSON rejected", contentType: "application/json", body: `{"title":"one"}{"title":"two"}`, wantStatus: http.StatusBadRequest},
		{name: "media type is case sensitive", contentType: "Application/JSON", body: `{}`, wantErr: ErrUnsupportedMediaType, wantStatus: http.StatusUnsupportedMediaType},
		{name: "null accepted", contentType: "application/json", body: `null`, want: payload{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			var got payload
			err := BindJSON(req, &got)
			if tt.wantErr != nil || tt.wantStatus != 0 {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
					t.Errorf("err: got %v, want %v", err, tt.wantErr)
				}
				if status := StatusCode(err); status != tt.wantStatus {
					t.Errorf("status code: got %d, want %d (%v)", status, tt.wantStatus, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("bound payload: got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestJSONBinder_ChunkedWithoutContentType(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{"title":"Hello"}`))
	req.ContentLength = -1 // unknown length, as with Transfer-Encoding: chunked

	var dst struct {
		Title string `json:"title"`
	}
	if err := BindJSON(req, &dst); !errors.Is(err, ErrUnsupportedMediaType) {
		t.Errorf("err: got %v, want %v", err, ErrUnsupportedMediaType)
	}
}

func TestJSONBinder_EmptyPreservesTarget(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/?title=query", nil)
	dst := struct {
		Title string `json:"title"`
	}{Title: "original"}
	if err := BindJSON(r, &dst); err != nil {
		t.Fatal(err)
	}
	if dst.Title != "original" {
		t.Fatalf("target changed: %+v", dst)
	}
}

func TestJSONBinder_UnknownLengthEmpty(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(""))
	req.ContentLength = -1
	req.Header.Set("Content-Type", "application/json")
	var dst any
	if err := BindJSON(req, &dst); StatusCode(err) != http.StatusBadRequest {
		t.Fatalf("error: %v, want 400", err)
	}
}

func TestJSONBinder_ReadErrors(t *testing.T) {
	t.Parallel()
	cause := errors.New("read failed")
	for _, tt := range []struct {
		name   string
		cause  error
		status int
		code   string
	}{
		{"read failure", cause, http.StatusBadRequest, erresp.CodeBinding},
		{"body limit", &http.MaxBytesError{Limit: 8}, http.StatusRequestEntityTooLarge, erresp.CodePayloadTooLarge},
		{"wrapped body limit", fmt.Errorf("read: %w", &http.MaxBytesError{Limit: 8}), http.StatusRequestEntityTooLarge, erresp.CodePayloadTooLarge},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)
			req.ContentLength = -1
			req.Header.Set("Content-Type", "application/json")
			req.Body = io.NopCloser(failingReader{err: tt.cause})
			var dst any
			err := BindJSON(req, &dst)
			typed, ok := errors.AsType[*StatusError](err)
			if !ok || typed.Code != tt.status || typed.Response.Code != tt.code {
				t.Fatalf("error: %#v, want %d %s", typed, tt.status, tt.code)
			}
			if !errors.Is(err, tt.cause) {
				t.Fatalf("cause lost: %v", err)
			}
			if tt.status == http.StatusRequestEntityTooLarge {
				limitErr, ok := errors.AsType[*http.MaxBytesError](err)
				if !ok || limitErr.Limit != 8 || !strings.Contains(err.Error(), "8-byte limit") {
					t.Fatalf("body limit lost: %v", err)
				}
			}
		})
	}
}

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }
