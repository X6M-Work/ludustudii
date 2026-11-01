package apispec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Bundle loads the specification and returns it as a single, self-contained
// JSON document, indented and ending in a newline, as package apidocs serves
// it.
func Bundle() ([]byte, error) {
	doc, err := Load()
	if err != nil {
		return nil, fmt.Errorf("load spec: %w", err)
	}
	b, err := bundle(doc)
	if err != nil {
		return nil, fmt.Errorf("bundle spec: %w", err)
	}
	return b, nil
}

// bundle moves every component doc references from another file under
// #/components and returns the result as JSON. It modifies doc. A component
// keeps the name it has in openapi.yml; others are named after their key in
// their file. A reference into a component, such as
// "#/LearningCardInput/properties/title", stays a JSON pointer into that
// component rather than becoming a component of its own. The result is parsed
// again without external references and validated.
func bundle(doc *openapi3.T) ([]byte, error) {
	if doc.Components == nil {
		doc.Components = &openapi3.Components{}
	}
	names := newRefNames(doc.Components)
	doc.InternalizeRefs(context.Background(), names.resolve)
	if names.err != nil {
		return nil, names.err
	}

	// InternalizeRefs points top-level headers and responses that reference
	// another file at themselves. Their values are still loaded, so dropping
	// the self-reference marshals the value in place.
	c := doc.Components
	errs := []error{
		dropPointers("schemas", c.Schemas),
		dropPointers("parameters", c.Parameters),
		dropPointers("headers", c.Headers),
		dropPointers("requestBodies", c.RequestBodies),
		dropPointers("responses", c.Responses),
		dropPointers("securitySchemes", c.SecuritySchemes),
		dropPointers("examples", c.Examples),
		dropPointers("links", c.Links),
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	derefSelf("schemas", c.Schemas, func(r *openapi3.SchemaRef) *string { return &r.Ref })
	derefSelf("parameters", c.Parameters, func(r *openapi3.ParameterRef) *string { return &r.Ref })
	derefSelf("headers", c.Headers, func(r *openapi3.HeaderRef) *string { return &r.Ref })
	derefSelf("requestBodies", c.RequestBodies, func(r *openapi3.RequestBodyRef) *string { return &r.Ref })
	derefSelf("responses", c.Responses, func(r *openapi3.ResponseRef) *string { return &r.Ref })
	derefSelf("securitySchemes", c.SecuritySchemes, func(r *openapi3.SecuritySchemeRef) *string { return &r.Ref })
	derefSelf("examples", c.Examples, func(r *openapi3.ExampleRef) *string { return &r.Ref })
	derefSelf("links", c.Links, func(r *openapi3.LinkRef) *string { return &r.Ref })

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	b = append(b, '\n')

	loader := openapi3.NewLoader()
	bundled, err := loader.LoadFromData(b)
	if err != nil {
		return nil, fmt.Errorf("parse bundled spec: %w", err)
	}
	if err := bundled.Validate(loader.Context, openapi3.EnableSchemaFormatValidation()); err != nil {
		return nil, fmt.Errorf("validate bundled spec: %w", err)
	}
	return b, nil
}

// refNames names internalized components and reports two different targets
// that would share a name.
type refNames struct {
	byTarget map[string]string // collection + " " + target -> name
	byName   map[string]string // collection + " " + name -> target
	err      error
}

// newRefNames seeds the names of the components declared in openapi.yml, so
// references to the same targets reuse them.
func newRefNames(c *openapi3.Components) *refNames {
	n := &refNames{byTarget: make(map[string]string), byName: make(map[string]string)}
	seedNames(n, "schemas", c.Schemas)
	seedNames(n, "parameters", c.Parameters)
	seedNames(n, "headers", c.Headers)
	seedNames(n, "requestBodies", c.RequestBodies)
	seedNames(n, "responses", c.Responses)
	seedNames(n, "securitySchemes", c.SecuritySchemes)
	seedNames(n, "examples", c.Examples)
	seedNames(n, "links", c.Links)
	return n
}

func (n *refNames) resolve(_ *openapi3.T, ref openapi3.ComponentRef) string {
	collection, u := ref.CollectionName(), ref.RefPath()
	top, rest, nested := strings.Cut(strings.Trim(u.Fragment, "/"), "/")
	if !nested {
		return n.name(collection, u)
	}
	// InternalizeRefs refers to the result as "#/components/<collection>/" +
	// name, which makes it a JSON pointer into the enclosing component. The
	// entry it also adds under this name is removed by dropPointers.
	parent := *u
	parent.Fragment = "/" + top
	return n.name(collection, &parent) + "/" + rest
}

func (n *refNames) name(collection string, u *url.URL) string {
	target := u.String()
	if name, ok := n.byTarget[collection+" "+target]; ok {
		return name
	}

	name := strings.Trim(u.Fragment, "/")
	if name == "" {
		name = strings.TrimSuffix(path.Base(u.Path), path.Ext(u.Path))
	}
	if other, ok := n.byName[collection+" "+name]; ok && other != target {
		n.err = errors.Join(n.err, fmt.Errorf("%s component name %q used by both %s and %s", collection, name, other, target))
	}
	n.byTarget[collection+" "+target] = name
	n.byName[collection+" "+name] = target
	return name
}

type componentRef interface {
	RefString() string
	RefPath() *url.URL
}

// seedNames visits components in name order, so when several reference the
// same target, the first name wins on every run and the bundle is stable.
func seedNames[R componentRef](n *refNames, collection string, components map[string]R) {
	for _, name := range slices.Sorted(maps.Keys(components)) {
		target := "#/components/" + collection + "/" + name
		if ref := components[name]; ref.RefString() != "" {
			target = ref.RefPath().String()
			if _, ok := n.byTarget[collection+" "+target]; !ok {
				n.byTarget[collection+" "+target] = name
			}
		}
		n.byName[collection+" "+name] = target
	}
}

// dropPointers removes the entries InternalizeRefs adds for JSON pointers into
// other components, and reports pointers whose component was not bundled.
func dropPointers[V any](collection string, components map[string]*V) error {
	var errs []error
	for name := range components {
		parent, _, ok := strings.Cut(name, "/")
		if !ok {
			continue
		}
		delete(components, name)
		if _, ok := components[parent]; !ok {
			errs = append(errs, fmt.Errorf("%s reference %q points into %q, which is not a component", collection, name, parent))
		}
	}
	return errors.Join(errs...)
}

func derefSelf[V any](collection string, components map[string]*V, ref func(*V) *string) {
	for name, v := range components {
		if r := ref(v); *r == "#/components/"+collection+"/"+name {
			*r = ""
		}
	}
}
