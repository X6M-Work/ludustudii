//go:build unit

package docs_test

import (
	"fmt"
	"testing"

	"github.com/apotourlyan/ludustudii/docs"
	"github.com/getkin/kin-openapi/openapi3"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	doc, err := docs.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	for _, p := range []string{"/api/lcards", "/api/lcards/{id}"} {
		if doc.Paths.Value(p) == nil {
			t.Errorf("path %s not documented", p)
		}
	}
}

// Every operation has a unique operationId, every GET has a documented HEAD,
// object schemas reachable from responses are closed, and object schemas
// reachable from request bodies deliberately tolerate unknown fields.
func TestContractConventions(t *testing.T) {
	t.Parallel()
	doc, err := docs.Load()
	if err != nil {
		t.Fatal(err)
	}
	ids := make(map[string]string)
	for path, item := range doc.Paths.Map() {
		// Echo AutoHandleHEAD exposes HEAD for every GET operation.
		if item.Get != nil && item.Head == nil {
			t.Errorf("%s has GET but no documented HEAD", path)
		}
		for method, op := range item.Operations() {
			if op.OperationID == "" {
				t.Errorf("%s %s missing operationId", method, path)
				continue
			}
			if previous, exists := ids[op.OperationID]; exists {
				t.Errorf("duplicate operationId %q: %s and %s %s", op.OperationID, previous, method, path)
			}
			ids[op.OperationID] = method + " " + path
		}
	}

	inputs := make(map[*openapi3.Schema]bool)
	outputs := make(map[*openapi3.Schema]bool)
	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			if op.RequestBody != nil {
				for mediaType, media := range op.RequestBody.Value.Content {
					where := fmt.Sprintf("%s %s request %s", method, path, mediaType)
					walkObjects(where, media.Schema, inputs, func(where string, s *openapi3.Schema) {
						if s.AdditionalProperties.Has != nil || s.AdditionalProperties.Schema != nil {
							t.Errorf("%s should permit unknown input fields", where)
						}
					})
				}
			}
			for status, resp := range op.Responses.Map() {
				for mediaType, media := range resp.Value.Content {
					where := fmt.Sprintf("%s %s %s %s", method, path, status, mediaType)
					walkObjects(where, media.Schema, outputs, func(where string, s *openapi3.Schema) {
						if s.AdditionalProperties.Has == nil || *s.AdditionalProperties.Has || s.AdditionalProperties.Schema != nil {
							t.Errorf("%s must declare additionalProperties: false", where)
						}
					})
				}
			}
		}
	}
}

// walkObjects calls check once for every object schema reachable from ref,
// labelling each with the property path that first reached it. seen guards
// against cycles. The loader resolves each file reference separately, so a
// shared schema can still be reported once per operation that uses it.
func walkObjects(where string, ref *openapi3.SchemaRef, seen map[*openapi3.Schema]bool, check func(string, *openapi3.Schema)) {
	if ref == nil || ref.Value == nil || seen[ref.Value] {
		return
	}
	s := ref.Value
	seen[s] = true
	if s.Type.Is(openapi3.TypeObject) {
		check(where, s)
	}
	for name, prop := range s.Properties {
		walkObjects(where+"."+name, prop, seen, check)
	}
	walkObjects(where+"[]", s.Items, seen, check)
	walkObjects(where+"{}", s.AdditionalProperties.Schema, seen, check)
	walkObjects(where+" not", s.Not, seen, check)
	for _, refs := range []openapi3.SchemaRefs{s.AllOf, s.AnyOf, s.OneOf} {
		for _, sub := range refs {
			walkObjects(where, sub, seen, check)
		}
	}
}
