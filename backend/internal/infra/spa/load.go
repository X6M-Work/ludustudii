// Package spa serves the single-page client application embedded from the
// dist directory, which the frontend build populates.
package spa

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/apotourlyan/ludustudii/internal/kit/staticfile"
)

//go:embed all:dist
var spaFS embed.FS

// ClientApp is the embedded client build, rooted at dist.
type ClientApp struct {
	RootFS    fs.FS
	IndexFile staticfile.File
}

// Load returns the embedded client app. It fails if dist has no index.html.
func Load() (*ClientApp, error) {
	rootFS, err := fs.Sub(spaFS, "dist")
	if err != nil {
		return nil, fmt.Errorf("create sub filesystem: %w", err)
	}

	indexBytes, err := fs.ReadFile(rootFS, "index.html")
	if err != nil {
		return nil, fmt.Errorf("read index.html: %w", err)
	}

	return &ClientApp{
		RootFS:    rootFS,
		IndexFile: staticfile.New("text/html; charset=utf-8", indexBytes),
	}, nil
}
