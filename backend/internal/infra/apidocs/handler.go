package apidocs

import (
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/kit/staticfile"
	"github.com/labstack/echo/v5"
)

// Path is where the reference page is served. The specification and the
// Scalar script are served below it, and the page loads them by URLs relative
// to Path, so dist/index.html names them after Path's last segment.
const Path = "/docs"

// Handler serves the reference page, the bundled specification and the Scalar
// script for GET and HEAD requests. None of them is content-hashed in its URL,
// so all are revalidated with their ETags.
type Handler struct {
	docs *Docs
}

// NewHandler returns a Handler that serves docs.
func NewHandler(docs *Docs) *Handler {
	return &Handler{docs: docs}
}

// RegisterRoutes registers the reference routes on e. Register them before
// the SPA catch-all, which would otherwise serve the client app for Path+"/".
// Other methods get the same plain-text 405 as the SPA rather than the JSON
// error of the API.
func (h *Handler) RegisterRoutes(e *echo.Echo) {
	e.GET(Path, serveFile(h.docs.Page))
	e.GET(Path+"/", redirectToPath)
	e.GET(Path+"/openapi.json", serveFile(h.docs.Spec))
	e.GET(Path+"/scalar.js", serveFile(h.docs.Script))
	for _, p := range []string{Path, Path + "/", Path + "/openapi.json", Path + "/scalar.js"} {
		e.Any(p, methodNotAllowed)
	}
}

// methodNotAllowed answers methods other than GET and HEAD. Echo routes a
// method-specific route ahead of an Any route for the same path.
func methodNotAllowed(c *echo.Context) error {
	c.Response().Header().Set("Allow", "GET, HEAD")
	return c.String(http.StatusMethodNotAllowed, "method not allowed\n")
}

// redirectToPath redirects Path+"/" to Path, keeping the query. The redirect
// is temporary so browsers do not keep it once the reference is disabled.
func redirectToPath(c *echo.Context) error {
	target := Path
	if q := c.Request().URL.RawQuery; q != "" {
		target += "?" + q
	}
	return c.Redirect(http.StatusFound, target)
}

func serveFile(f staticfile.File) echo.HandlerFunc {
	return func(c *echo.Context) error {
		f.Serve(c.Response(), c.Request())
		return nil
	}
}
