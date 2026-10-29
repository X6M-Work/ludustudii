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

type ClientApp struct {
	RootFS    fs.FS
	IndexFile IndexFile
}

type IndexFile struct {
	ETag       string
	IndexBytes []byte
}

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
