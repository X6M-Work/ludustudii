//go:build apidocs

package apidocs

import (
	"embed"
	"errors"
	"io/fs"
)

// assetsFS holds dist/assets/scalar.js.gz, the gzipped Scalar standalone
// bundle that the Makefile writes when the frontend dependencies are
// installed. The committed .gitkeep lets this compile without it.
//
//go:embed all:dist/assets
var assetsFS embed.FS

// readScript returns the gzipped Scalar bundle, or nil if it was not written
// before the build.
func readScript() ([]byte, error) {
	b, err := assetsFS.ReadFile("dist/assets/scalar.js.gz")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return b, err
}
