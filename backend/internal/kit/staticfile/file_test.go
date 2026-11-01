//go:build unit

package staticfile

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNew_ETag(t *testing.T) {
	t.Parallel()

	a, b := New("text/plain", []byte("a")), New("text/plain", []byte("b"))
	if !strings.HasPrefix(a.ETag, `"`) || !strings.HasSuffix(a.ETag, `"`) {
		t.Errorf("ETag %s is not a quoted strong ETag", a.ETag)
	}
	if a.ETag == b.ETag {
		t.Errorf("different contents share ETag %s", a.ETag)
	}
	if again := New("text/html", []byte("a")); again.ETag != a.ETag {
		t.Errorf("same content: got ETag %s, want %s", again.ETag, a.ETag)
	}
}

func TestFile_Serve(t *testing.T) {
	t.Parallel()
	f := New("text/javascript; charset=utf-8", []byte("console.log(1)"))

	t.Run("GET", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		f.Serve(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d, want 200", rec.Code)
		}
		for header, value := range map[string]string{
			"Content-Type":  f.ContentType,
			"Cache-Control": "no-cache",
			"ETag":          f.ETag,
		} {
			if got := rec.Header().Get(header); got != value {
				t.Errorf("%s: got %q, want %q", header, got, value)
			}
		}
		if !bytes.Equal(rec.Body.Bytes(), f.Bytes) {
			t.Error("body differs from the preloaded file")
		}
	})

	t.Run("If-None-Match", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("If-None-Match", f.ETag)
		rec := httptest.NewRecorder()
		f.Serve(rec, req)

		if rec.Code != http.StatusNotModified {
			t.Errorf("status: got %d, want 304", rec.Code)
		}
	})
}

func gzipped(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(b); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestNewGzip_RejectsInvalid(t *testing.T) {
	t.Parallel()

	gz := gzipped(t, []byte("console.log(1)"))
	for name, b := range map[string][]byte{
		"empty":     nil,
		"plain":     []byte("console.log(1)"),
		"truncated": gz[:len(gz)-4],
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewGzip("text/javascript", b); err == nil {
				t.Error("NewGzip() succeeded, want error")
			}
		})
	}
}

func TestFile_ServeGzip(t *testing.T) {
	t.Parallel()
	plain := []byte("console.log(1)")
	f, err := NewGzip("text/javascript; charset=utf-8", gzipped(t, plain))
	if err != nil {
		t.Fatal(err)
	}

	for name, tt := range map[string]struct {
		acceptEncoding string
		wantEncoding   string
		wantBody       []byte
	}{
		"accepts gzip":              {acceptEncoding: "gzip, deflate, br", wantEncoding: "gzip", wantBody: f.Bytes},
		"no gzip":                   {acceptEncoding: "", wantEncoding: "", wantBody: plain},
		"refuses gzip":              {acceptEncoding: "*, gzip;q=0", wantEncoding: "", wantBody: plain},
		"accepts any":               {acceptEncoding: "br, *;q=0.5", wantEncoding: "gzip", wantBody: f.Bytes},
		"accepts x-gzip":            {acceptEncoding: "x-gzip", wantEncoding: "gzip", wantBody: f.Bytes},
		"refuses gzip, q not first": {acceptEncoding: "gzip;level=1;Q=0", wantEncoding: "", wantBody: plain},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			rec := httptest.NewRecorder()
			f.Serve(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status: got %d, want 200", rec.Code)
			}
			if got := rec.Header().Get("Content-Encoding"); got != tt.wantEncoding {
				t.Errorf("Content-Encoding: got %q, want %q", got, tt.wantEncoding)
			}
			if got := rec.Header().Get("Vary"); got != "Accept-Encoding" {
				t.Errorf("Vary: got %q, want Accept-Encoding", got)
			}
			if !bytes.Equal(rec.Body.Bytes(), tt.wantBody) {
				t.Errorf("body: got %q, want %q", rec.Body.Bytes(), tt.wantBody)
			}

			// Each representation revalidates with its own ETag.
			etag := rec.Header().Get("ETag")
			if (etag == f.ETag) != (tt.wantEncoding == "gzip") {
				t.Errorf("ETag %s: want the gzip ETag %s only for the gzip body", etag, f.ETag)
			}
			req = httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			req.Header.Set("If-None-Match", etag)
			rec = httptest.NewRecorder()
			f.Serve(rec, req)
			if rec.Code != http.StatusNotModified {
				t.Errorf("If-None-Match: status got %d, want 304", rec.Code)
			}
		})
	}
}

// A range of the gzip body cannot be decompressed on its own, so it is always
// sent whole; the decompressed body still supports ranges.
func TestFile_ServeGzipRange(t *testing.T) {
	t.Parallel()
	plain := []byte("console.log(1)")
	f, err := NewGzip("text/javascript; charset=utf-8", gzipped(t, plain))
	if err != nil {
		t.Fatal(err)
	}

	for name, tt := range map[string]struct {
		acceptEncoding string
		wantStatus     int
		wantRanges     string
		wantBody       []byte
	}{
		"gzip":     {acceptEncoding: "gzip", wantStatus: http.StatusOK, wantRanges: "none", wantBody: f.Bytes},
		"identity": {acceptEncoding: "", wantStatus: http.StatusPartialContent, wantRanges: "bytes", wantBody: plain[:4]},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			req.Header.Set("Range", "bytes=0-3")
			rec := httptest.NewRecorder()
			f.Serve(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Accept-Ranges"); got != tt.wantRanges {
				t.Errorf("Accept-Ranges: got %q, want %q", got, tt.wantRanges)
			}
			if !bytes.Equal(rec.Body.Bytes(), tt.wantBody) {
				t.Errorf("body: got %q, want %q", rec.Body.Bytes(), tt.wantBody)
			}
			if req.Header.Get("Range") == "" {
				t.Error("Serve() removed the Range header from the caller's request")
			}
		})
	}
}
