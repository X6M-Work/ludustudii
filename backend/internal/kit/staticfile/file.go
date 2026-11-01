// Package staticfile serves preloaded response bodies that are revalidated
// with a strong ETag derived from their content.
package staticfile

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// File is a preloaded response body with a strong ETag derived from its
// content.
type File struct {
	ContentType string
	ETag        string
	Bytes       []byte
	// Encoding is "gzip" when Bytes is gzip-compressed, otherwise empty.
	Encoding string

	// decoded is Bytes decompressed. It is set by NewGzip.
	decoded []byte
}

// New returns a File for b served as contentType.
func New(contentType string, b []byte) File {
	hash := sha256.Sum256(b)
	return File{
		ContentType: contentType,
		ETag:        `"` + hex.EncodeToString(hash[:16]) + `"`,
		Bytes:       b,
	}
}

// NewGzip returns a File for gz, the gzip-compressed content served as
// contentType. Clients that accept gzip get gz as is; the others get a copy
// decompressed once here, which also checks that gz is valid gzip.
func NewGzip(contentType string, gz []byte) (File, error) {
	decoded, err := gunzip(gz)
	if err != nil {
		return File{}, fmt.Errorf("decompress: %w", err)
	}
	f := New(contentType, gz)
	f.Encoding = "gzip"
	f.decoded = decoded
	return f, nil
}

func gunzip(gz []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return b, r.Close()
}

// Serve writes f for GET and HEAD requests with Cache-Control "no-cache", so
// clients revalidate it with its ETag and get 304 when it is unchanged. A
// gzip-compressed File is sent decompressed, under its own ETag, to clients
// that do not accept gzip. The compressed body is always sent whole, since a
// range of it cannot be decompressed on its own.
func (f File) Serve(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Content-Type", f.ContentType)
	h.Set("Cache-Control", "no-cache")

	body, etag := f.Bytes, f.ETag
	if f.Encoding != "" {
		h.Add("Vary", "Accept-Encoding")
		if accepts(r.Header.Get("Accept-Encoding"), f.Encoding) {
			h.Set("Content-Encoding", f.Encoding)
			w, r = noRanges{w}, withoutRange(r)
		} else {
			body, etag = f.decoded, strings.TrimSuffix(etag, `"`)+`-identity"`
		}
	}
	h.Set("ETag", etag)
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(body))
}

// noRanges overrides the "Accept-Ranges: bytes" that http.ServeContent sets.
type noRanges struct{ http.ResponseWriter }

func (w noRanges) WriteHeader(code int) {
	w.Header().Set("Accept-Ranges", "none")
	w.ResponseWriter.WriteHeader(code)
}

func (w noRanges) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// withoutRange returns r without its Range header, leaving r unchanged.
func withoutRange(r *http.Request) *http.Request {
	if r.Header.Get("Range") == "" {
		return r
	}
	c := *r
	c.Header = r.Header.Clone()
	c.Header.Del("Range")
	return &c
}

// accepts reports whether an Accept-Encoding header value allows coding with a
// nonzero quality. An entry naming coding, or its "x-" alias, takes
// precedence over "*".
func accepts(header, coding string) bool {
	star := false
	for part := range strings.SplitSeq(header, ",") {
		name, params, _ := strings.Cut(part, ";")
		name = strings.TrimSpace(name)
		allowed := true
		for param := range strings.SplitSeq(params, ";") {
			key, value, _ := strings.Cut(param, "=")
			if !strings.EqualFold(strings.TrimSpace(key), "q") {
				continue
			}
			if v, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && v == 0 {
				allowed = false
			}
		}
		switch {
		case strings.EqualFold(name, coding), strings.EqualFold(name, "x-"+coding):
			return allowed
		case name == "*":
			star = allowed
		}
	}
	return star
}
