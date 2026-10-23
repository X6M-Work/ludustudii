package spa

import (
	"bytes"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"
)

type Handler struct {
	fileServer http.Handler
	rootFS     fs.FS
	indexFile  IndexFile
}

func NewHandler(clientApp *ClientApp) *Handler {
	return &Handler{
		fileServer: http.FileServerFS(clientApp.RootFS),
		rootFS:     clientApp.RootFS,
		indexFile:  clientApp.IndexFile,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	reqPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")

	if reqPath == "" || reqPath == "index.html" {
		serveIndexFile(w, r, h.indexFile)
		return
	}

	stat, err := fs.Stat(h.rootFS, reqPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Do not serve index.html for missing static assets (.js, .css, .png, etc.)
			if path.Ext(reqPath) != "" {
				http.NotFound(w, r)
				return
			}

			serveIndexFile(w, r, h.indexFile)
			return
		}

		slog.ErrorContext(r.Context(), "Failed to stat path",
			slog.String("url", r.URL.Path),
			slog.String("path", reqPath),
			slog.Any("error", err),
		)

		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if stat.IsDir() {
		serveIndexFile(w, r, h.indexFile)
		return
	}

	if strings.HasPrefix(reqPath, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}

	h.fileServer.ServeHTTP(w, r)
}

func serveIndexFile(w http.ResponseWriter, r *http.Request, f IndexFile) {
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("ETag", f.ETag)
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(f.IndexBytes))
}
