//go:build unit

package testkit

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/docs"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

func TestCheckResponse(t *testing.T) {
	t.Parallel()

	const validID = "019a0000-0000-7000-8000-000000000001"
	const cardBody = `{"id":"` + validID + `","title":"OpenAPI","description":"Contract test"}`

	jsonHeaders := func(extra ...string) http.Header {
		if len(extra)%2 != 0 {
			panic("jsonHeaders: odd number of key/value arguments")
		}
		h := http.Header{
			"Content-Type": {"application/json"},
			"X-Request-Id": {"req-1"},
		}
		for i := 0; i < len(extra); i += 2 {
			h.Set(extra[i], extra[i+1])
		}
		return h
	}

	// wantErr is a substring of the expected error; empty means no error.
	tests := []struct {
		name    string
		method  string
		path    string
		status  int
		headers http.Header
		body    string
		wantErr string
	}{
		{
			name:    "valid GET card",
			method:  http.MethodGet,
			path:    "/api/lcards/" + validID,
			status:  http.StatusOK,
			headers: jsonHeaders(),
			body:    cardBody,
		},
		{
			name:    "undeclared field in response fails additionalProperties",
			method:  http.MethodGet,
			path:    "/api/lcards/" + validID,
			status:  http.StatusOK,
			headers: jsonHeaders(),
			body:    `{"id":"` + validID + `","title":"OpenAPI","description":"Contract test","extra":true}`,
			wantErr: `property "extra" is unsupported`,
		},
		{
			name:    "undeclared field in error response fails additionalProperties",
			method:  http.MethodGet,
			path:    "/api/lcards/" + validID,
			status:  http.StatusNotFound,
			headers: jsonHeaders(),
			body:    `{"code":"not_found","message":"nope"}`,
			wantErr: `property "message" is unsupported`,
		},
		{
			name:    "missing X-Request-Id header fails",
			method:  http.MethodGet,
			path:    "/api/lcards/" + validID,
			status:  http.StatusOK,
			headers: http.Header{"Content-Type": {"application/json"}},
			body:    cardBody,
			wantErr: `header "X-Request-Id" missing`,
		},
		{
			name:    "wrong Content-Type fails",
			method:  http.MethodGet,
			path:    "/api/lcards/" + validID,
			status:  http.StatusOK,
			headers: jsonHeaders("Content-Type", "text/plain"),
			body:    cardBody,
			wantErr: `Content-Type has unexpected value: "text/plain"`,
		},
		{
			name:    "missing Content-Type fails",
			method:  http.MethodGet,
			path:    "/api/lcards/" + validID,
			status:  http.StatusOK,
			headers: http.Header{"X-Request-Id": {"req-1"}},
			body:    cardBody,
			wantErr: `Content-Type has unexpected value: ""`,
		},
		{
			name:    "undocumented status code fails",
			method:  http.MethodPost,
			path:    "/api/lcards",
			status:  http.StatusOK,
			headers: jsonHeaders(),
			body:    `{"id":"` + validID + `"}`,
			wantErr: "status 200 is not documented",
		},
		{
			name:    "valid 201 POST with Location",
			method:  http.MethodPost,
			path:    "/api/lcards",
			status:  http.StatusCreated,
			headers: jsonHeaders("Location", "/api/lcards/"+validID),
			body:    `{"id":"` + validID + `"}`,
		},
		{
			name:    "201 POST missing Location header fails",
			method:  http.MethodPost,
			path:    "/api/lcards",
			status:  http.StatusCreated,
			headers: jsonHeaders(),
			body:    `{"id":"` + validID + `"}`,
			wantErr: `header "Location" missing`,
		},
		{
			name:    "non-empty body on 204 fails",
			method:  http.MethodDelete,
			path:    "/api/lcards/" + validID,
			status:  http.StatusNoContent,
			headers: http.Header{"X-Request-Id": {"req-1"}},
			body:    `{}`,
			wantErr: "status 204 declares no content",
		},
		{
			name:    "valid HEAD",
			method:  http.MethodHead,
			path:    "/api/lcards",
			status:  http.StatusOK,
			headers: http.Header{"X-Request-Id": {"req-1"}},
		},
		{
			name:    "HEAD missing X-Request-Id fails",
			method:  http.MethodHead,
			path:    "/api/lcards",
			status:  http.StatusOK,
			headers: http.Header{},
			wantErr: `header "X-Request-Id" missing`,
		},
		{
			name:    "HEAD with body fails",
			method:  http.MethodHead,
			path:    "/api/lcards",
			status:  http.StatusOK,
			headers: http.Header{"X-Request-Id": {"req-1"}},
			body:    `x`,
			wantErr: "status 200 declares no content",
		},
		{
			name:    "undocumented API endpoint returning 200 fails",
			method:  http.MethodGet,
			path:    "/api/undocumented",
			status:  http.StatusOK,
			headers: jsonHeaders(),
			body:    `{}`,
			wantErr: "undocumented API path returned 200",
		},
		{
			name:    "undocumented API endpoint returning 404 passes",
			method:  http.MethodGet,
			path:    "/api/undocumented",
			status:  http.StatusNotFound,
			headers: jsonHeaders(),
			body:    `{"code":"not_found"}`,
		},
		{
			name:    "undocumented API endpoint returning 500 passes",
			method:  http.MethodGet,
			path:    "/api/undocumented",
			status:  http.StatusInternalServerError,
			headers: jsonHeaders(),
			body:    `{"code":"unexpected_error"}`,
		},
		{
			name:    "undocumented API endpoint 404 with invalid code fails",
			method:  http.MethodGet,
			path:    "/api/undocumented",
			status:  http.StatusNotFound,
			headers: jsonHeaders(),
			body:    `{"code":"nope"}`,
			wantErr: "value is not one of the allowed values",
		},
		{
			name:    "undocumented API endpoint 404 missing X-Request-Id fails",
			method:  http.MethodGet,
			path:    "/api/undocumented",
			status:  http.StatusNotFound,
			headers: http.Header{"Content-Type": {"application/json"}},
			body:    `{"code":"not_found"}`,
			wantErr: `header "X-Request-Id" missing`,
		},
		{
			name:    "undocumented API endpoint HEAD 404 passes",
			method:  http.MethodHead,
			path:    "/api/undocumented",
			status:  http.StatusNotFound,
			headers: http.Header{"X-Request-Id": {"req-1"}},
		},
		{
			name:    "undocumented API endpoint HEAD 404 with body fails",
			method:  http.MethodHead,
			path:    "/api/undocumented",
			status:  http.StatusNotFound,
			headers: http.Header{"X-Request-Id": {"req-1"}},
			body:    `x`,
			wantErr: "HEAD returned 1-byte body, want empty",
		},
		{
			name:    "valid 405 with Allow",
			method:  http.MethodPatch,
			path:    "/api/lcards",
			status:  http.StatusMethodNotAllowed,
			headers: jsonHeaders("Allow", "GET, HEAD, POST"),
			body:    `{"code":"method_not_allowed"}`,
		},
		{
			name:    "405 missing Allow header fails",
			method:  http.MethodPatch,
			path:    "/api/lcards",
			status:  http.StatusMethodNotAllowed,
			headers: jsonHeaders(),
			body:    `{"code":"method_not_allowed"}`,
			wantErr: "405 response missing Allow header",
		},
		{
			name:    "undocumented method not returning 405 fails",
			method:  http.MethodPatch,
			path:    "/api/lcards",
			status:  http.StatusOK,
			headers: jsonHeaders(),
			body:    `{}`,
			wantErr: "undocumented method returned 200, want 405",
		},
		{
			name:    "405 with undeclared field fails",
			method:  http.MethodPatch,
			path:    "/api/lcards",
			status:  http.StatusMethodNotAllowed,
			headers: jsonHeaders("Allow", "GET, HEAD, POST"),
			body:    `{"code":"method_not_allowed","extra":true}`,
			wantErr: `property "extra" is unsupported`,
		},
		{
			name:    "undocumented 301 fails",
			method:  http.MethodGet,
			path:    "/api/lcards/" + validID,
			status:  http.StatusMovedPermanently,
			headers: http.Header{},
			body:    `not json`,
			wantErr: "status 301 is not documented",
		},
		{
			name:    "undocumented 304 fails",
			method:  http.MethodGet,
			path:    "/api/lcards/" + validID,
			status:  http.StatusNotModified,
			headers: http.Header{},
			wantErr: "status 304 is not documented",
		},
		{
			name:    "undocumented 307 fails",
			method:  http.MethodGet,
			path:    "/api/lcards/" + validID,
			status:  http.StatusTemporaryRedirect,
			headers: http.Header{},
			body:    `not json`,
			wantErr: "status 307 is not documented",
		},
		{
			name:    "undocumented 308 fails",
			method:  http.MethodGet,
			path:    "/api/lcards/" + validID,
			status:  http.StatusPermanentRedirect,
			headers: http.Header{},
			body:    `not json`,
			wantErr: "status 308 is not documented",
		},
		{
			name:    "HEAD undocumented 304 fails",
			method:  http.MethodHead,
			path:    "/api/lcards",
			status:  http.StatusNotModified,
			headers: http.Header{"X-Request-Id": {"req-1"}},
			wantErr: "status 304 is not documented",
		},
		{
			name:    "trailing newline after JSON passes",
			method:  http.MethodGet,
			path:    "/api/lcards/" + validID,
			status:  http.StatusOK,
			headers: jsonHeaders(),
			body:    cardBody + "\n",
		},
		{
			name:    "trailing garbage after JSON fails",
			method:  http.MethodGet,
			path:    "/api/lcards/" + validID,
			status:  http.StatusOK,
			headers: jsonHeaders(),
			body:    cardBody + "GARBAGE",
			wantErr: "body is not a single JSON value",
		},
		{
			name:    "concatenated JSON objects fail",
			method:  http.MethodGet,
			path:    "/api/lcards/" + validID,
			status:  http.StatusOK,
			headers: jsonHeaders(),
			body:    cardBody + cardBody,
			wantErr: "body is not a single JSON value",
		},
		{
			name:    "concatenated JSON objects on undocumented API path fail",
			method:  http.MethodGet,
			path:    "/api/undocumented",
			status:  http.StatusNotFound,
			headers: jsonHeaders(),
			body:    `{"code":"not_found"}{"code":"not_found"}`,
			wantErr: "body is not a single JSON value",
		},
		{
			name:    "non-API SPA route is ignored",
			method:  http.MethodGet,
			path:    "/",
			status:  http.StatusOK,
			headers: http.Header{"Content-Type": {"text/html; charset=utf-8"}},
			body:    `<!doctype html>`,
		},
		{
			name:    "valid page with null cursor",
			method:  http.MethodGet,
			path:    "/api/lcards",
			status:  http.StatusOK,
			headers: jsonHeaders(),
			body:    `{"items":[` + cardBody + `],"next_cursor":null}`,
		},
		{
			name:    "valid page with UUIDv7 cursor",
			method:  http.MethodGet,
			path:    "/api/lcards",
			status:  http.StatusOK,
			headers: jsonHeaders(),
			body:    `{"items":[` + cardBody + `],"next_cursor":"019a0000-0000-7000-8000-000000000001"}`,
		},
		{
			name:    "page rejects extra property",
			method:  http.MethodGet,
			path:    "/api/lcards",
			status:  http.StatusOK,
			headers: jsonHeaders(),
			body:    `{"items":[],"next_cursor":null,"extra":true}`,
			wantErr: `property "extra" is unsupported`,
		},
		{
			name:    "created card rejects extra property",
			method:  http.MethodPost,
			path:    "/api/lcards",
			status:  http.StatusCreated,
			headers: jsonHeaders("Location", "/api/lcards/"+validID),
			body:    `{"id":"` + validID + `","extra":true}`,
			wantErr: `property "extra" is unsupported`,
		},
		{
			name:    "valid field error",
			method:  http.MethodPost,
			path:    "/api/lcards",
			status:  http.StatusBadRequest,
			headers: jsonHeaders(),
			body:    `{"code":"validation_error","errors":[{"field":"title","code":"required"}]}`,
		},
		{
			name:    "field error rejects extra property",
			method:  http.MethodPost,
			path:    "/api/lcards",
			status:  http.StatusBadRequest,
			headers: jsonHeaders(),
			body:    `{"code":"validation_error","errors":[{"field":"title","code":"required","extra":true}]}`,
			wantErr: `property "extra" is unsupported`,
		},
		{
			name:    "empty field errors fail minItems",
			method:  http.MethodPost,
			path:    "/api/lcards",
			status:  http.StatusBadRequest,
			headers: jsonHeaders(),
			body:    `{"code":"validation_error","errors":[]}`,
			wantErr: `minimum number of items is 1`,
		},
		{
			name:    "page rejects uppercase UUIDv7 cursor",
			method:  http.MethodGet,
			path:    "/api/lcards",
			status:  http.StatusOK,
			headers: jsonHeaders(),
			body:    `{"items":[],"next_cursor":"019A0000-0000-7000-8000-000000000001"}`,
			wantErr: `string doesn't match the regular expression`,
		},
		{
			name:    "page rejects uppercase UUIDv7 card ID",
			method:  http.MethodGet,
			path:    "/api/lcards",
			status:  http.StatusOK,
			headers: jsonHeaders(),
			body:    `{"items":[{"id":"019A0000-0000-7000-8000-000000000001","title":"OpenAPI","description":"Contract test"}],"next_cursor":null}`,
			wantErr: `string doesn't match the regular expression`,
		},
		{
			name:    "page rejects UUIDv4 cursor",
			method:  http.MethodGet,
			path:    "/api/lcards",
			status:  http.StatusOK,
			headers: jsonHeaders(),
			body:    `{"items":[],"next_cursor":"019a0000-0000-4000-8000-000000000001"}`,
			wantErr: `string doesn't match the regular expression`,
		},
		{
			name:    "page rejects UUIDv4 card ID",
			method:  http.MethodGet,
			path:    "/api/lcards",
			status:  http.StatusOK,
			headers: jsonHeaders(),
			body:    `{"items":[{"id":"019a0000-0000-4000-8000-000000000001","title":"OpenAPI","description":"Contract test"}],"next_cursor":null}`,
			wantErr: `string doesn't match the regular expression`,
		},
		{
			name:    "empty JSON body fails",
			method:  http.MethodGet,
			path:    "/api/lcards/" + validID,
			status:  http.StatusOK,
			headers: jsonHeaders(),
			body:    "",
			wantErr: `failed to decode response body: EOF`,
		},
		{
			name:    "whitespace JSON body fails",
			method:  http.MethodGet,
			path:    "/api/lcards/" + validID,
			status:  http.StatusOK,
			headers: jsonHeaders(),
			body:    "   ",
			wantErr: `body is not a single JSON value`,
		},
		{
			name:    "405 missing X-Request-Id fails",
			method:  http.MethodPatch,
			path:    "/api/lcards",
			status:  http.StatusMethodNotAllowed,
			headers: http.Header{"Content-Type": {"application/json"}, "Allow": {"GET, HEAD, POST"}},
			body:    `{"code":"method_not_allowed"}`,
			wantErr: `header "X-Request-Id" missing`,
		},
		{
			name:    "bare /api returning 200 fails",
			method:  http.MethodGet,
			path:    "/api",
			status:  http.StatusOK,
			headers: http.Header{"Content-Type": {"text/html; charset=utf-8"}},
			body:    `<!doctype html>`,
			wantErr: "undocumented API path returned 200",
		},
		{
			name:    "non-API path sharing the /api prefix is ignored",
			method:  http.MethodGet,
			path:    "/apix",
			status:  http.StatusOK,
			headers: http.Header{"Content-Type": {"text/html; charset=utf-8"}},
			body:    `<!doctype html>`,
		},
		{
			name:    "path below a card returning 404 passes",
			method:  http.MethodGet,
			path:    "/api/lcards/" + validID + "/deeper",
			status:  http.StatusNotFound,
			headers: jsonHeaders(),
			body:    `{"code":"not_found"}`,
		},
		{
			name:    "path below a card returning 200 fails",
			method:  http.MethodGet,
			path:    "/api/lcards/" + validID + "/deeper",
			status:  http.StatusOK,
			headers: jsonHeaders(),
			body:    `{}`,
			wantErr: "undocumented API path returned 200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tt.method, "http://example.com"+tt.path, nil)
			resp := &http.Response{
				StatusCode: tt.status,
				Header:     tt.headers,
			}
			err := checkResponse(req, resp, []byte(tt.body))
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("checkResponse() unexpected error: %v", err)
			case tt.wantErr != "" && err == nil:
				t.Fatalf("checkResponse() = nil, want error containing %q", tt.wantErr)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Fatalf("checkResponse() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

// Production documents HEAD wherever GET is available (Echo AutoHandleHEAD).
// A local spec without HEAD exercises the unsupported-method fallback itself.
func TestCheckUnmatchedRouteHEADMethodNotAllowed(t *testing.T) {
	t.Parallel()
	doc, err := docs.Load()
	if err != nil {
		t.Fatal(err)
	}
	doc.Paths.Value("/api/lcards").Head = nil
	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodHead, "http://example.com/api/lcards", nil)
	_, _, routeErr := router.FindRoute(req)
	if !errors.Is(routeErr, routers.ErrMethodNotAllowed) {
		t.Fatalf("FindRoute() = %v, want ErrMethodNotAllowed", routeErr)
	}
	for _, tt := range []struct {
		name    string
		headers http.Header
		body    string
		wantErr string
	}{
		{name: "valid", headers: http.Header{"Allow": {"GET, POST"}, "X-Request-Id": {"req-1"}}},
		{name: "missing request ID", headers: http.Header{"Allow": {"GET, POST"}}, wantErr: `header "X-Request-Id" missing`},
		{name: "missing Allow", headers: http.Header{"X-Request-Id": {"req-1"}}, wantErr: "405 response missing Allow header"},
		{name: "body", headers: http.Header{"Allow": {"GET, POST"}, "X-Request-Id": {"req-1"}}, body: "x", wantErr: "HEAD returned 1-byte body, want empty"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := &http.Response{StatusCode: http.StatusMethodNotAllowed, Header: tt.headers}
			err := checkUnmatchedRoute(doc, req, resp, []byte(tt.body), routeErr)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("checkUnmatchedRoute() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestDocumentedResponse(t *testing.T) {
	t.Parallel()
	exact := &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription("exact")}
	ranged := &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription("range")}
	fallback := &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription("default")}
	withDefault := openapi3.NewResponses()
	withDefault.Set("404", exact)
	withDefault.Set("4XX", ranged)
	withDefault.Set("default", fallback)
	withoutDefault := openapi3.NewResponses()
	withoutDefault.Set("404", exact)

	for _, tt := range []struct {
		name      string
		responses *openapi3.Responses
		status    int
		want      *openapi3.ResponseRef
	}{
		{name: "exact", responses: withDefault, status: 404, want: exact},
		{name: "range", responses: withDefault, status: 400, want: ranged},
		{name: "default", responses: withDefault, status: 500, want: fallback},
		{name: "undocumented", responses: withoutDefault, status: 500, want: nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := documentedResponse(tt.responses, tt.status); got != tt.want {
				t.Fatalf("documentedResponse(%d) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

type errorRecorder struct {
	testing.TB
	errors []string
}

func (r *errorRecorder) Helper() {}

func (r *errorRecorder) Errorf(format string, args ...any) {
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

func TestDoDoesNotFollowRedirects(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/lcards" {
			http.Redirect(w, r, "/other", http.StatusMovedPermanently)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req-1")
		_, _ = w.Write([]byte(`{"items":[],"next_cursor":null}`))
	}))
	t.Cleanup(srv.Close)

	rec := &errorRecorder{TB: t}
	resp, _ := Do(rec, srv, DoOpts{Method: http.MethodGet, Path: "/api/lcards"})
	if resp.StatusCode != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusMovedPermanently)
	}
	if len(rec.errors) != 1 || !strings.Contains(rec.errors[0], "status 301 is not documented") {
		t.Fatalf("errors = %v, want 1 error containing %q", rec.errors, "status 301 is not documented")
	}
}
