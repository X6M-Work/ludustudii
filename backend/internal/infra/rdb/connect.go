// Package rdb connects to the relational (PostgreSQL) database.
package rdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Options configures the connection pool opened by Connect. The pool fields
// are passed to the matching *sql.DB setters.
type Options struct {
	ConnectionString string
	MaxOpenConns     int
	MaxIdleConns     int
	ConnMaxLifetime  time.Duration
	ConnMaxIdleTime  time.Duration
}

// Connect opens a pgx-backed *sql.DB for opts.ConnectionString, configures
// its pool and pings the database, giving up after 5 seconds. The caller must
// close the returned DB.
func Connect(opts Options) (*sql.DB, error) {
	db, err := sql.Open("pgx", opts.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(opts.MaxOpenConns)
	db.SetMaxIdleConns(opts.MaxIdleConns)
	db.SetConnMaxLifetime(opts.ConnMaxLifetime)
	db.SetConnMaxIdleTime(opts.ConnMaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}
