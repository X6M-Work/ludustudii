// Package migrations embeds the goose SQL migrations for the database schema.
// See goose.md for the command-line workflow.
package migrations

import (
	"context"
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
)

// FS holds the migration files.
//
//go:embed *.sql
var FS embed.FS

// Up applies all pending migrations in FS to db.
func Up(ctx context.Context, db *sql.DB) error {
	p, err := goose.NewProvider(goose.DialectPostgres, db, FS)
	if err != nil {
		return err
	}
	_, err = p.Up(ctx)
	return err
}
