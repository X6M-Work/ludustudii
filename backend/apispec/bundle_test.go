//go:build unit

package apispec

import (
	"encoding/json"
	"maps"
	"net/url"
	"path"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/getkin/kin-openapi/openapi3"
)

// The bundled specification is self-contained, has exactly the components
// declared in openapi.yml, and describes exactly what the multi-file
// specification does once every reference is replaced by its target.
func TestBundleSpec(t *testing.T) {
	t.Parallel()

	doc, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	b, err := bundle(doc)
	if err != nil {
		t.Fatalf("bundle() error: %v", err)
	}

	var bundled map[string]any
	if err := json.Unmarshal(b, &bundled); err != nil {
		t.Fatal(err)
	}
	components, _ := bundled["components"].(map[string]any)

	original, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for collection, names := range map[string][]string{
		"schemas":       keys(original.Components.Schemas),
		"parameters":    keys(original.Components.Parameters),
		"headers":       keys(original.Components.Headers),
		"requestBodies": keys(original.Components.RequestBodies),
		"responses":     keys(original.Components.Responses),
	} {
		got, _ := components[collection].(map[string]any)
		if len(got) != len(names) {
			t.Errorf("components.%s: got %v, want exactly %v", collection, slices.Sorted(maps.Keys(got)), slices.Sorted(slices.Values(names)))
		}
		for _, name := range names {
			c, _ := got[name].(map[string]any)
			if c == nil {
				t.Errorf("components.%s.%s missing", collection, name)
			} else if _, ok := c["$ref"]; ok {
				t.Errorf("components.%s.%s is a reference, want its value: %v", collection, name, c)
			}
		}
	}

	clearRefs(reflect.ValueOf(original), make(map[uintptr]bool))
	ob, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]any
	if err := json.Unmarshal(ob, &want); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"openapi", "info", "servers", "security", "tags", "paths"} {
		got := inlineRefs(t, bundled[key], components, 0)
		if !reflect.DeepEqual(got, want[key]) {
			gb, _ := json.MarshalIndent(got, "", "  ")
			wb, _ := json.MarshalIndent(want[key], "", "  ")
			t.Errorf("%s differs after inlining references\ngot:  %s\nwant: %s", key, gb, wb)
		}
	}
}

func TestBundleSpec_NameCollision(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"openapi.yml": {Data: []byte(`openapi: 3.0.3
info: {title: t, version: "1"}
paths:
  /a:
    get:
      responses:
        "200":
          description: a
          content:
            application/json:
              schema: {$ref: './a.yml#/Thing'}
  /b:
    get:
      responses:
        "200":
          description: b
          content:
            application/json:
              schema: {$ref: './b.yml#/Thing'}
`)},
		"a.yml": {Data: []byte("Thing: {type: string}\n")},
		"b.yml": {Data: []byte("Thing: {type: integer}\n")},
	}

	_, err := bundle(loadFS(t, fsys))
	if err == nil || !strings.Contains(err.Error(), `"Thing"`) {
		t.Errorf("bundle() error: got %v, want a collision on \"Thing\"", err)
	}
}

func TestBundleSpec_PointerIntoNonComponent(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"openapi.yml": {Data: []byte(`openapi: 3.0.3
info: {title: t, version: "1"}
paths:
  /a:
    get:
      responses:
        "200":
          description: a
          content:
            application/json:
              schema: {$ref: './a.yml#/Thing/properties/name'}
`)},
		"a.yml": {Data: []byte("Thing: {type: object, properties: {name: {type: string}}}\n")},
	}

	_, err := bundle(loadFS(t, fsys))
	if err == nil || !strings.Contains(err.Error(), `"Thing"`) {
		t.Errorf("bundle() error: got %v, want one naming \"Thing\"", err)
	}
}

// loadFS loads openapi.yml from fsys, resolving file references within it.
func loadFS(t *testing.T, fsys fstest.MapFS) *openapi3.T {
	t.Helper()
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(_ *openapi3.Loader, uri *url.URL) ([]byte, error) {
		return fsys.ReadFile(strings.TrimPrefix(path.Clean(uri.Path), "/"))
	}
	doc, err := loader.LoadFromFile("openapi.yml")
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// clearRefs empties every exported string field named Ref reachable from v,
// so marshalling a loaded document inlines every reference target. Types such
// as Paths and Responses keep their entries in an unexported map, reached
// through their Map method.
func clearRefs(v reflect.Value, seen map[uintptr]bool) {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() || seen[v.Pointer()] {
			return
		}
		seen[v.Pointer()] = true
		if m := v.MethodByName("Map"); m.IsValid() && m.Type().NumIn() == 0 &&
			m.Type().NumOut() == 1 && m.Type().Out(0).Kind() == reflect.Map {
			clearRefs(m.Call(nil)[0], seen)
		}
		clearRefs(v.Elem(), seen)
	case reflect.Interface:
		if !v.IsNil() {
			clearRefs(v.Elem(), seen)
		}
	case reflect.Struct:
		for i := range v.NumField() {
			f, sf := v.Field(i), v.Type().Field(i)
			if !sf.IsExported() {
				continue
			}
			if sf.Name == "Ref" && f.Kind() == reflect.String {
				f.SetString("")
				continue
			}
			clearRefs(f, seen)
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			clearRefs(v.Index(i), seen)
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			clearRefs(v.MapIndex(k), seen)
		}
	}
}

// inlineRefs replaces every {"$ref": "#/components/<collection>/<name>"} in
// node with the referenced component, recursively. The name may continue as a
// JSON pointer into the component.
func inlineRefs(t *testing.T, node any, components map[string]any, depth int) any {
	t.Helper()
	if depth > 64 {
		t.Fatal("reference cycle")
	}
	switch n := node.(type) {
	case map[string]any:
		if ref, ok := n["$ref"].(string); ok {
			var target any = components
			for _, seg := range strings.Split(strings.TrimPrefix(ref, "#/components/"), "/") {
				m, _ := target.(map[string]any)
				target = m[strings.NewReplacer("~1", "/", "~0", "~").Replace(seg)]
			}
			if !strings.HasPrefix(ref, "#/components/") || target == nil {
				t.Fatalf("unresolvable $ref %q", ref)
			}
			return inlineRefs(t, target, components, depth+1)
		}
		out := make(map[string]any, len(n))
		for k, v := range n {
			out[k] = inlineRefs(t, v, components, depth+1)
		}
		return out
	case []any:
		out := make([]any, len(n))
		for i, v := range n {
			out[i] = inlineRefs(t, v, components, depth+1)
		}
		return out
	default:
		return node
	}
}
