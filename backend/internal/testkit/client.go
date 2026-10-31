package testkit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

// DoOpts describes a request sent by Do. Path is appended to the server URL;
// empty ContentType and Body are not sent. Chunked sends Body without a
// Content-Length, using chunked transfer encoding. Headers are copied before
// ContentType is applied; a nonempty ContentType overrides that header.
type DoOpts struct {
	Method      string
	Path        string
	ContentType string
	Headers     http.Header
	Body        string
	Chunked     bool
}

// Do sends the request described by o to s, with a 10-second timeout, without
// following redirects, validates the response against the OpenAPI
// specification, and returns the response together with its fully read body.
// The request itself is not validated, so tests can send invalid input.
// It fails the test if the request cannot be sent or the body cannot be read.
func Do(t testing.TB, s *httptest.Server, o DoOpts) (*http.Response, []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var r io.Reader
	if o.Body != "" {
		r = strings.NewReader(o.Body)
		if o.Chunked {
			// Hiding the reader's type stops NewRequest from setting ContentLength.
			r = io.MultiReader(r)
		}
	}
	req, err := http.NewRequestWithContext(ctx, o.Method, s.URL+o.Path, r)
	if err != nil {
		t.Fatal(err)
	}
	if o.Headers != nil {
		req.Header = o.Headers.Clone()
	}
	if o.ContentType != "" {
		req.Header.Set("Content-Type", o.ContentType)
	}
	client := *s.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	ValidateResponse(t, req, resp, content)
	return resp, content
}

// AssertStatus fails the test unless resp has status code want. The failure
// message includes body.
func AssertStatus(t *testing.T, resp *http.Response, body []byte, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("status: got %d, want %d\n%s", resp.StatusCode, want, body)
	}
}

// AssertJSON compares JSON structurally against a literal, so the test pins
// the wire format (field names, [] vs null) instead of reusing handler types.
func AssertJSON(t *testing.T, got []byte, want string) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("response is not JSON: %v\n%s", err, got)
	}
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("bad want literal: %v", err)
	}
	if !reflect.DeepEqual(g, w) {
		t.Errorf("body:\n got %s\nwant %s", got, want)
	}
}
