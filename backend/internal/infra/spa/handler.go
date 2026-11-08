package spa

import (
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"

	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
	"github.com/apotourlyan/ludustudii/internal/kit/staticfile"
)

// Handler serves the client app for GET and HEAD requests. Paths that name a
// file serve that file; files under assets/ are content-hashed by the build
// and cached as immutable, others are revalidated. The root, directories and
// extensionless paths that match no file serve index.html so the client-side
// router can handle them. Missing paths with an extension get 404.
type Handler struct {
	fileServer http.Handler
	rootFS     fs.FS
	indexFile  staticfile.File
	onError    httpkit.ErrorHandler
}

// NewHandler returns a Handler that serves clientApp and reports errors to
// onError. It needs neither the database nor the API.
func NewHandler(clientApp *ClientApp, onError httpkit.ErrorHandler) *Handler {
	return &Handler{
		fileServer: http.FileServerFS(clientApp.RootFS),
		rootFS:     clientApp.RootFS,
		indexFile:  clientApp.IndexFile,
		onError:    onError,
	}
}

// Mount registers the client application's catch-all route on mux, so every
// path no other surface claims serves the client app. The pattern is
// method-less because "GET /" conflicts with method-less prefixes of other
// surfaces, such as "/api/"; serve checks the method itself.
func (h *Handler) Mount(mux *http.ServeMux) {
	mux.Handle("/", httpkit.Adapt(h.serve, h.onError))
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		return httpkit.Text(w, http.StatusMethodNotAllowed, "method not allowed\n")
	}

	reqPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")

	if reqPath == "" || reqPath == "index.html" {
		h.indexFile.Serve(w, r)
		return nil
	}

	stat, err := fs.Stat(h.rootFS, reqPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Do not serve index.html for missing static assets (.js, .css, .png, etc.)
			if path.Ext(reqPath) != "" {
				return httpkit.Text(w, http.StatusNotFound, "404 page not found\n")
			}

			h.indexFile.Serve(w, r)
			return nil
		}

		slog.ErrorContext(r.Context(), "Failed to stat path",
			slog.String("url", r.URL.Path),
			slog.String("path", reqPath),
			slog.Any("error", err),
		)

		return httpkit.Text(w, http.StatusInternalServerError, "Internal Server Error\n")
	}

	if stat.IsDir() {
		h.indexFile.Serve(w, r)
		return nil
	}

	if strings.HasPrefix(reqPath, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}

	h.fileServer.ServeHTTP(w, r)
	return nil
}
