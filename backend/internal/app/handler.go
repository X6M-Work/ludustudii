package app

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/app/api"
	"github.com/apotourlyan/ludustudii/internal/app/router"
	"github.com/apotourlyan/ludustudii/internal/infra/apidocs"
	"github.com/apotourlyan/ludustudii/internal/infra/health"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
)

// parts are what this binary serves. Run loads them from the environment;
// tests build them directly.
type parts struct {
	db        *sql.DB
	readiness *health.State
	clientApp *spa.ClientApp
	docs      *apidocs.Docs // nil when the API reference is disabled
}

// newHandler builds each surface and mounts it on the shared router. Each
// surface owns its patterns and depends only on its own part:
//
//	/livez, /readyz   probes (health), readiness state only
//	/api, /api/...    JSON API (api), database only
//	GET /docs/...     API reference (apidocs), only when enabled
//	/                 client app catch-all (spa), embedded files only
//
// Middleware errors use the JSON error shape on every surface.
func newHandler(log *slog.Logger, p parts) http.Handler {
	onError := router.ErrorHandler(log)
	surfaces := []router.Surface{
		health.NewProbes(p.readiness, onError),
		api.New(p.db, onError),
	}
	if p.docs != nil {
		surfaces = append(surfaces, apidocs.NewHandler(p.docs))
	}
	surfaces = append(surfaces, spa.NewHandler(p.clientApp, onError))
	return router.New(log, onError, surfaces...)
}
