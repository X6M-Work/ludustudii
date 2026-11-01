//go:build unit

package apidocs

import (
	"bytes"
	"compress/gzip"
	"errors"
	"path"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/apispec"
	"github.com/apotourlyan/ludustudii/internal/kit/staticfile"
)

// gzipped returns b gzip-compressed, as the Makefile embeds the Scalar script.
func gzipped(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(b); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// Without the apidocs tag, or before the frontend dependencies are installed,
// the binary has no Scalar script.
func TestLoad(t *testing.T) {
	t.Parallel()

	_, err := Load()
	script, readErr := readScript()
	switch {
	case readErr != nil:
		if !errors.Is(err, ErrBuiltWithoutDocs) {
			t.Fatalf("Load() without apidocs tag: got error %v, want ErrBuiltWithoutDocs", err)
		}
	case len(script) == 0:
		if !errors.Is(err, ErrScriptNotEmbedded) {
			t.Fatalf("Load() without script: got error %v, want ErrScriptNotEmbedded", err)
		}
	case err != nil:
		t.Fatalf("Load() error: %v", err)
	}
}

func TestLoad_RejectsMissingScript(t *testing.T) {
	t.Parallel()

	if _, err := load([]byte("<!doctype html>"), nil); !errors.Is(err, ErrScriptNotEmbedded) {
		t.Errorf("load() without script: got error %v, want ErrScriptNotEmbedded", err)
	}
	if _, err := load([]byte("<!doctype html>"), []byte("console.log(1)")); err == nil {
		t.Error("load() with a script that is not gzipped succeeded, want error")
	}
}

// The page is served at Path and loads its files by URLs relative to it, which
// resolve below Path only while they start with Path's last segment.
func TestPage_RelativeURLs(t *testing.T) {
	t.Parallel()

	page, err := docsFS.ReadFile("dist/index.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"openapi.json", "scalar.js"} {
		if rel := path.Base(Path) + "/" + file; !bytes.Contains(page, []byte(rel)) {
			t.Errorf("page does not load %s", rel)
		}
	}
}

func TestLoad_ETags(t *testing.T) {
	t.Parallel()

	d, err := load([]byte("<!doctype html>"), gzipped(t, []byte("console.log(1)")))
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, f := range []staticfile.File{d.Page, d.Spec, d.Script} {
		if !strings.HasPrefix(f.ETag, `"`) || !strings.HasSuffix(f.ETag, `"`) {
			t.Errorf("ETag %s is not a quoted strong ETag", f.ETag)
		}
		if seen[f.ETag] {
			t.Errorf("duplicate ETag %s", f.ETag)
		}
		seen[f.ETag] = true
	}
}

// dist/openapi.json is what apispec.Bundle produces from the current
// specification.
func TestBundledSpec(t *testing.T) {
	t.Parallel()

	want, err := apispec.Bundle()
	if err != nil {
		t.Fatal(err)
	}
	got, err := docsFS.ReadFile("dist/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Error("dist/openapi.json is stale: run 'go generate ./internal/infra/apidocs' in backend")
	}
}
