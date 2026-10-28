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

type DoOpts struct {
	Method      string
	Path        string
	ContentType string
	Body        string
}

func Do(t *testing.T, s *httptest.Server, o DoOpts) (*http.Response, []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var r io.Reader
	if o.Body != "" {
		r = strings.NewReader(o.Body)
	}
	req, err := http.NewRequestWithContext(ctx, o.Method, s.URL+o.Path, r)
	if err != nil {
		t.Fatal(err)
	}
	if o.ContentType != "" {
		req.Header.Set("Content-Type", o.ContentType)
	}
	resp, err := s.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, content
}

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
