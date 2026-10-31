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

	"github.com/labstack/echo/v5"
)

// Handler serves the client app for GET and HEAD requests. Paths that name a
// file serve that file; files under assets/ are content-hashed by the build
// and cached as immutable, others are revalidated. The root, directories and
// extensionless paths that match no file serve index.html so the client-side
// router can handle them. Missing paths with an extension get 404.
type Handler struct {
	fileServer http.Handler
	rootFS     fs.FS
	indexFile  IndexFile
}

// NewHandler returns a Handler that serves clientApp.
func NewHandler(clientApp *ClientApp) *Handler {
	return &Handler{
		fileServer: http.FileServerFS(clientApp.RootFS),
		rootFS:     clientApp.RootFS,
		indexFile:  clientApp.IndexFile,
	}
}

// RegisterRoutes registers the handler as e's catch-all for paths no other
// route matches. Register it after all other routes.
func (h *Handler) RegisterRoutes(e *echo.Echo) {
	e.RouteNotFound("/*", h.serve)
}

func (h *Handler) serve(c *echo.Context) error {
	w, r := c.Response(), c.Request()
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		return c.String(http.StatusMethodNotAllowed, "method not allowed\n")
	}

	reqPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")

	if reqPath == "" || reqPath == "index.html" {
		serveIndexFile(w, r, h.indexFile)
		return nil
	}

	stat, err := fs.Stat(h.rootFS, reqPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Do not serve index.html for missing static assets (.js, .css, .png, etc.)
			if path.Ext(reqPath) != "" {
				return c.String(http.StatusNotFound, "404 page not found\n")
			}

			serveIndexFile(w, r, h.indexFile)
			return nil
		}

		slog.ErrorContext(r.Context(), "Failed to stat path",
			slog.String("url", r.URL.Path),
			slog.String("path", reqPath),
			slog.Any("error", err),
		)

		return c.String(http.StatusInternalServerError, "Internal Server Error\n")
	}

	if stat.IsDir() {
		serveIndexFile(w, r, h.indexFile)
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

func serveIndexFile(w http.ResponseWriter, r *http.Request, f IndexFile) {
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("ETag", f.ETag)
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(f.IndexBytes))
}
