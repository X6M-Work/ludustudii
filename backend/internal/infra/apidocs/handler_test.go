//go:build unit

package apidocs

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/kit/staticfile"
)

func newTestServer(t *testing.T) (http.Handler, *Docs) {
	t.Helper()
	d, err := load([]byte("<!doctype html>"), gzipped(t, []byte("console.log(1)")))
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	NewHandler(d).Mount(mux)
	return mux, d
}

func TestHandler(t *testing.T) {
	t.Parallel()
	e, d := newTestServer(t)

	for path, want := range map[string]staticfile.File{
		Path:                   d.Page,
		Path + "/openapi.json": d.Spec,
		Path + "/scalar.js":    d.Script,
	} {
		t.Run("GET "+path, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
			req.Header.Set("Accept-Encoding", "gzip")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status: got %d, want 200", rec.Code)
			}
			for header, value := range map[string]string{
				"Content-Type":     want.ContentType,
				"Cache-Control":    "no-cache",
				"ETag":             want.ETag,
				"Content-Encoding": want.Encoding,
			} {
				if got := rec.Header().Get(header); got != value {
					t.Errorf("%s: got %q, want %q", header, got, value)
				}
			}
			if !bytes.Equal(rec.Body.Bytes(), want.Bytes) {
				t.Error("body differs from the preloaded file")
			}
		})

		t.Run("HEAD "+path, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodHead, path, nil))

			if rec.Code != http.StatusOK {
				t.Fatalf("status: got %d, want 200", rec.Code)
			}
			if rec.Body.Len() != 0 {
				t.Errorf("HEAD body: got %d bytes, want 0", rec.Body.Len())
			}
		})

		t.Run("GET "+path+" If-None-Match", func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
			req.Header.Set("Accept-Encoding", "gzip")
			req.Header.Set("If-None-Match", want.ETag)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotModified {
				t.Errorf("status: got %d, want 304", rec.Code)
			}
		})
	}
}

func TestHandler_TrailingSlashRedirects(t *testing.T) {
	t.Parallel()
	e, _ := newTestServer(t)

	for target, want := range map[string]string{
		Path + "/":       Path,
		Path + "/?x=1&y": Path + "?x=1&y",
	} {
		t.Run(target, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil))

			if rec.Code != http.StatusFound {
				t.Fatalf("status: got %d, want 302", rec.Code)
			}
			if got := rec.Header().Get("Location"); got != want {
				t.Errorf("Location: got %q, want %q", got, want)
			}
		})
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	e, _ := newTestServer(t)

	for _, path := range []string{Path, Path + "/", Path + "/openapi.json", Path + "/scalar.js"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, nil))

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status: got %d, want 405", rec.Code)
			}
			if got := rec.Header().Get("Allow"); got != "GET, HEAD" {
				t.Errorf("Allow: got %q, want %q", got, "GET, HEAD")
			}
			if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
				t.Errorf("Content-Type: got %q, want text/plain", got)
			}
		})
	}
}
