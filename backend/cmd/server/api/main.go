// Command api runs the ludustudii HTTP server: the JSON API under /api and the
// embedded single-page client application. It is configured through
// environment variables; see package env.
package main

import (
	"log/slog"
	"os"

	"github.com/apotourlyan/ludustudii/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		slog.Error("Application terminated with error", "error", err)
		os.Exit(1)
	}
}
