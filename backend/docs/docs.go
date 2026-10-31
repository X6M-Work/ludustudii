// Package docs embeds and loads the OpenAPI specification for the HTTP API.
package docs

import (
	"embed"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// FS holds the OpenAPI specification files rooted at openapi.yml.
//
//go:embed openapi.yml paths components
var FS embed.FS

// Load parses and validates the embedded OpenAPI specification, resolving all
// relative file references across FS.
func Load() (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(_ *openapi3.Loader, uri *url.URL) ([]byte, error) {
		p := strings.TrimPrefix(path.Clean(uri.Path), "/")
		return FS.ReadFile(p)
	}

	doc, err := loader.LoadFromFile("openapi.yml")
	if err != nil {
		return nil, err
	}
	if err := doc.Validate(loader.Context, openapi3.EnableSchemaFormatValidation()); err != nil {
		return nil, err
	}
	return doc, nil
}
