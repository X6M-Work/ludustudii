package spa

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"
)

//go:embed all:dist
var spaFS embed.FS

func Handler() http.Handler {
	rootFS, err := fs.Sub(spaFS, "dist")
	if err != nil {
		panic(err)
	}

	indexBytes, err := fs.ReadFile(rootFS, "index.html")
	if err != nil {
		panic(err)
	}

	hash := sha256.Sum256(indexBytes)
	etag := `"` + hex.EncodeToString(hash[:16]) + `"`
	modifiedAt := time.Now().UTC()
	fileServer := http.FileServer(http.FS(rootFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(path.Clean(r.URL.Path), "/")

		if path == "" || path == "index.html" {
			serveIndexFile(w, r, etag, modifiedAt, indexBytes)
			return
		}

		stat, err := fs.Stat(rootFS, path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				serveIndexFile(w, r, etag, modifiedAt, indexBytes)
				return
			}

			slog.ErrorContext(r.Context(), "spa: failed to stat path",
				slog.String("url", r.URL.Path),
				slog.String("path", path),
				slog.Any("error", err),
			)

			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if stat.IsDir() {
			serveIndexFile(w, r, etag, modifiedAt, indexBytes)
			return
		}

		if strings.HasPrefix(path, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}

		fileServer.ServeHTTP(w, r)
	})
}

func serveIndexFile(w http.ResponseWriter, r *http.Request, etag string, modifiedAt time.Time, indexBytes []byte) {
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("ETag", etag)
	http.ServeContent(w, r, "index.html", modifiedAt, bytes.NewReader(indexBytes))
}
