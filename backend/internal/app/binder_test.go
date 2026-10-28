package app

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
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
			contentType: echo.MIMEApplicationJSON,
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
			contentType: echo.MIMEApplicationJSON,
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
			wantErr:     echo.ErrUnsupportedMediaType,
			wantStatus:  http.StatusUnsupportedMediaType,
		},
		{
			name:        "xml / non-empty / unsupported media type",
			contentType: echo.MIMEApplicationXML,
			body:        `<payload><Title>Hello</Title></payload>`,
			wantErr:     echo.ErrUnsupportedMediaType,
			wantStatus:  http.StatusUnsupportedMediaType,
		},
		{
			name:        "json / malformed / bad request",
			contentType: echo.MIMEApplicationJSON,
			body:        `{"title":`,
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := echo.NewWithConfig(echo.Config{Binder: jsonBinder{}})
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set(echo.HeaderContentType, tt.contentType)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			var got payload
			err := c.Bind(&got)
			if tt.wantErr != nil || tt.wantStatus != 0 {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
					t.Errorf("err: got %v, want %v", err, tt.wantErr)
				}
				if status := echo.StatusCode(err); status != tt.wantStatus {
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
	e := echo.NewWithConfig(echo.Config{Binder: jsonBinder{}})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"title":"Hello"}`))
	req.ContentLength = -1 // unknown length, as with Transfer-Encoding: chunked
	c := e.NewContext(req, httptest.NewRecorder())

	var dst struct {
		Title string `json:"title"`
	}
	if err := c.Bind(&dst); !errors.Is(err, echo.ErrUnsupportedMediaType) {
		t.Errorf("err: got %v, want %v", err, echo.ErrUnsupportedMediaType)
	}
}
