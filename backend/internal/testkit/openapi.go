package testkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/apotourlyan/ludustudii/docs"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

type openAPISpec struct {
	doc    *openapi3.T
	router routers.Router
}

var loadOpenAPI = sync.OnceValues(func() (*openAPISpec, error) {
	doc, err := docs.Load()
	if err != nil {
		return nil, err
	}
	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		return nil, err
	}
	return &openAPISpec{doc: doc, router: router}, nil
})

// ValidateResponse fails the test if resp and body do not conform to the
// OpenAPI specification in backend/docs for req. Non-API paths outside the
// spec (such as the SPA root) are ignored.
func ValidateResponse(t testing.TB, req *http.Request, resp *http.Response, body []byte) {
	t.Helper()
	if err := checkResponse(req, resp, body); err != nil {
		t.Errorf("OpenAPI (%s %s -> %d): %v", req.Method, req.URL.Path, resp.StatusCode, err)
	}
}

func checkResponse(req *http.Request, resp *http.Response, body []byte) error {
	spec, err := loadOpenAPI()
	if err != nil {
		return fmt.Errorf("load spec: %w", err)
	}

	route, pathParams, err := spec.router.FindRoute(req)
	if err != nil {
		return checkUnmatchedRoute(spec.doc, req, resp, body, err)
	}

	// openapi3filter.ValidateResponse skips some statuses (301, 304, 307, 308)
	// even with IncludeResponseStatus, so documented status is checked here.
	respRef := documentedResponse(route.Operation.Responses, resp.StatusCode)
	if respRef == nil || respRef.Value == nil {
		return fmt.Errorf("status %d is not documented", resp.StatusCode)
	}
	if len(respRef.Value.Content) == 0 && len(body) != 0 {
		return fmt.Errorf("status %d declares no content, got %d-byte body: %s",
			resp.StatusCode, len(body), body)
	}
	if err := checkSingleJSONValue(respRef.Value, body); err != nil {
		return err
	}

	// openapi3filter.ValidateResponse returns early when req.Method is HEAD,
	// skipping status and header checks. A shallow copy with Method set to GET
	// runs those checks against route.Operation (which is still the HEAD
	// operation, whose responses in head.yml have no content section).
	// A HEAD body is caught by the "declares no content" check above.
	vReq := req
	if req.Method == http.MethodHead {
		clone := *req
		clone.Method = http.MethodGet
		vReq = &clone
	}

	input := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    vReq,
			PathParams: pathParams,
			Route:      route,
		},
		Status: resp.StatusCode,
		Header: resp.Header,
		Body:   io.NopCloser(bytes.NewReader(body)),
		Options: &openapi3filter.Options{
			IncludeResponseStatus: true,
		},
	}
	return openapi3filter.ValidateResponse(req.Context(), input)
}

// documentedResponse returns the response documented for status, falling back
// from the exact code to its range (such as 4XX) and then to default, in the
// same order as openapi3filter.ValidateResponse.
func documentedResponse(responses *openapi3.Responses, status int) *openapi3.ResponseRef {
	if ref := responses.Status(status); ref != nil {
		return ref
	}
	return responses.Default()
}

func checkUnmatchedRoute(
	doc *openapi3.T,
	req *http.Request,
	resp *http.Response,
	body []byte,
	routeErr error,
) error {
	switch {
	case errors.Is(routeErr, routers.ErrPathNotFound):
		if req.URL.Path != "/api" && !strings.HasPrefix(req.URL.Path, "/api/") {
			return nil
		}
		if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusInternalServerError {
			return fmt.Errorf("undocumented API path returned %d", resp.StatusCode)
		}
	case errors.Is(routeErr, routers.ErrMethodNotAllowed):
		if resp.StatusCode != http.StatusMethodNotAllowed {
			return fmt.Errorf("undocumented method returned %d, want 405", resp.StatusCode)
		}
		if resp.Header.Get("Allow") == "" {
			return errors.New("405 response missing Allow header")
		}
	default:
		return fmt.Errorf("route lookup: %w", routeErr)
	}

	// Like documented errors, fallbacks share ErrorResponse without coupling
	// status to code. Handler integration tests pin codes and exact Allow values.
	// Fallback API errors (404, 405, 500) must still carry X-Request-Id and
	// conform to the ErrorResponse schema (or have an empty body for HEAD).
	notFoundRef := doc.Components.Responses["NotFound"]
	op := openapi3.NewOperation()
	if req.Method == http.MethodHead {
		if len(body) != 0 {
			return fmt.Errorf("HEAD returned %d-byte body, want empty", len(body))
		}
		// Only its headers are checked, so it also serves HEAD 405 and 500.
		headRef := doc.Components.Responses["HeadNotFound"]
		op.AddResponse(resp.StatusCode, headRef.Value)
	} else {
		if err := checkSingleJSONValue(notFoundRef.Value, body); err != nil {
			return err
		}
		op.AddResponse(resp.StatusCode, notFoundRef.Value)
	}

	vReq := req
	if req.Method == http.MethodHead {
		clone := *req
		clone.Method = http.MethodGet
		vReq = &clone
	}

	input := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request: vReq,
			Route: &routers.Route{
				Spec:      doc,
				Path:      req.URL.Path,
				Method:    req.Method,
				Operation: op,
			},
		},
		Status: resp.StatusCode,
		Header: resp.Header,
		Body:   io.NopCloser(bytes.NewReader(body)),
		Options: &openapi3filter.Options{
			IncludeResponseStatus: true,
		},
	}
	return openapi3filter.ValidateResponse(req.Context(), input)
}

// checkSingleJSONValue fails if response declares JSON content and body is not
// exactly one JSON value. openapi3filter decodes only the first value and
// ignores trailing data, such as a second body from a double write.
func checkSingleJSONValue(response *openapi3.Response, body []byte) error {
	if response.Content.Get("application/json") == nil || len(body) == 0 {
		return nil
	}
	if !json.Valid(body) {
		return fmt.Errorf("body is not a single JSON value: %s", body)
	}
	return nil
}
