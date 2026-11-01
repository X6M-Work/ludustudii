// Package apidocs serves an interactive API reference, rendered by Scalar from
// the OpenAPI specification in package apispec.
package apidocs

import (
	"embed"
	"errors"
	"fmt"

	"github.com/apotourlyan/ludustudii/internal/kit/staticfile"
)

// dist/openapi.json is the specification bundled by apispec.Bundle, so the
// server serves it without parsing it. TestBundledSpec fails when it is stale.
//
//go:generate go run github.com/apotourlyan/ludustudii/cmd/tools/bundlespec dist/openapi.json
//go:embed dist/index.html dist/openapi.json
var docsFS embed.FS

// ErrBuiltWithoutDocs reports that the binary was built without the apidocs
// tag, so it cannot serve the API reference.
var ErrBuiltWithoutDocs = errors.New("api reference not built in: build with -tags=apidocs (make build-backend API_DOCS=1)")

// ErrScriptNotEmbedded reports that the binary was built with the apidocs tag
// but before the frontend dependencies were installed, so it has no Scalar
// script.
var ErrScriptNotEmbedded = errors.New("scalar bundle not embedded: run 'npm install' in frontend and rebuild")

// Docs is the preloaded reference page, the bundled specification and the
// Scalar script that renders it.
type Docs struct {
	Page   staticfile.File
	Spec   staticfile.File
	Script staticfile.File
}

// Load returns the embedded reference page, bundled OpenAPI specification and
// gzipped Scalar script. It returns
// ErrBuiltWithoutDocs if the binary was built without the apidocs tag, and
// ErrScriptNotEmbedded if it has no Scalar script.
func Load() (*Docs, error) {
	page, err := docsFS.ReadFile("dist/index.html")
	if err != nil {
		return nil, fmt.Errorf("read dist/index.html: %w", err)
	}
	scriptGzip, err := readScript()
	if err != nil {
		return nil, err
	}
	return load(page, scriptGzip)
}

func load(page, scriptGzip []byte) (*Docs, error) {
	if len(scriptGzip) == 0 {
		return nil, ErrScriptNotEmbedded
	}
	script, err := staticfile.NewGzip("text/javascript; charset=utf-8", scriptGzip)
	if err != nil {
		return nil, fmt.Errorf("load scalar bundle: %w", err)
	}
	spec, err := docsFS.ReadFile("dist/openapi.json")
	if err != nil {
		return nil, fmt.Errorf("read dist/openapi.json: %w", err)
	}
	return &Docs{
		Page:   staticfile.New("text/html; charset=utf-8", page),
		Spec:   staticfile.New("application/json", spec),
		Script: script,
	}, nil
}
