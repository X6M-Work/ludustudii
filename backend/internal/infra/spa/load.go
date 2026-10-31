// Package spa serves the single-page client application embedded from the
// dist directory, which the frontend build populates.
package spa

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
)

//go:embed all:dist
var spaFS embed.FS

// ClientApp is the embedded client build, rooted at dist.
type ClientApp struct {
	RootFS    fs.FS
	IndexFile IndexFile
}

// IndexFile is the preloaded index.html with a strong ETag derived from its
// content.
type IndexFile struct {
	ETag       string
	IndexBytes []byte
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

	hash := sha256.Sum256(indexBytes)
	etag := `"` + hex.EncodeToString(hash[:16]) + `"`

	return &ClientApp{
		RootFS: rootFS,
		IndexFile: IndexFile{
			ETag:       etag,
			IndexBytes: indexBytes,
		},
	}, nil
}
