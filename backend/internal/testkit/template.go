package testkit

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

const dbConnectionStringKey = "DB_TEST_CONNECTION"
const dbTimeout = 10 * time.Second

// Template owns a migrated database used to create isolated copies.
// Clones are independent databases, but their cleanup functions use the template's
// administrative connection. Run all clone cleanup functions before calling Close,
// which shuts down that connection.
type Template struct {
	admin   *sql.DB
	config  *pgx.ConnConfig
	name    string
	cloneMu sync.Mutex
}

// NewTemplate creates and migrates a template using DB_TEST_CONNECTION.
// The configured role must be able to create databases and connect to postgres.
// The caller must close the template after all copies are closed.
// Setup, including migration, is limited to 10 seconds or the context deadline.
func NewTemplate(ctx context.Context, migrate func(context.Context, *sql.DB) error) (*Template, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()
	// Get the connection string from the environment.
	connection := os.Getenv(dbConnectionStringKey)
	if connection == "" {
		return nil, fmt.Errorf("get env %s: not set", dbConnectionStringKey)
	}
	// Parse the connection string into a pgx configuration.
	config, err := pgx.ParseConfig(connection)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", dbConnectionStringKey, err)
	}
	// Generate a unique name for the template database.
	name, err := generateName("tmpl_")
	if err != nil {
		return nil, err
	}
	// Connect to the administrative database and create the template database.
	admin := stdlib.OpenDB(*config)
	if _, err := admin.ExecContext(ctx, createSQL(name)); err != nil {
		return nil, errors.Join(fmt.Errorf("create template: %w", err), admin.Close())
	}
	template := &Template{admin: admin, config: config, name: name}
	// Open, migrate, and close the connection to the template database.
	tmplConfig := config.Copy()
	tmplConfig.Database = name
	db := stdlib.OpenDB(*tmplConfig)
	if err := errors.Join(migrate(ctx, db), db.Close()); err != nil {
		return nil, errors.Join(fmt.Errorf("migrate template: %w", err), template.Close())
	}
	// Terminate any lingering connections to the template database before cloning.
	if _, err := template.admin.ExecContext(ctx, terminateSQL, name); err != nil {
		return nil, errors.Join(fmt.Errorf("terminate template connections: %w", err), template.Close())
	}
	return template, nil
}

// Clone returns a connection to a fresh copy and a cleanup function that closes
// the connection and drops the copy. The caller must invoke cleanup when done.
// Clone may be called concurrently. Database creation is limited to 10 seconds
// or the context deadline. Cleanup uses an independent 10-second timeout.
func (t *Template) Clone(ctx context.Context) (*sql.DB, func() error, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()
	// Generate a unique name for the clone database.
	name, err := generateName("test_")
	if err != nil {
		return nil, nil, err
	}
	// Create the clone database using the template.
	t.cloneMu.Lock()
	_, err = t.admin.ExecContext(ctx, cloneSQL(name, t.name))
	t.cloneMu.Unlock()
	if err != nil {
		return nil, nil, fmt.Errorf("clone database: %w", err)
	}
	// Open a connection to the clone database.
	cloneConfig := t.config.Copy()
	cloneConfig.Database = name
	db := stdlib.OpenDB(*cloneConfig)
	// Create a cleanup function that closes the connection and drops the clone database.
	cleanup := func() error {
		closeErr := db.Close()
		ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
		defer cancel()
		_, dropErr := t.admin.ExecContext(ctx, dropSQL(name))
		if dropErr != nil {
			dropErr = fmt.Errorf("drop database: %w", dropErr)
		}
		return errors.Join(closeErr, dropErr)
	}
	return db, cleanup, nil
}

// Close drops the template and closes its administrative connection.
// It uses an independent 10-second timeout for dropping the database.
// It must not run concurrently with Clone or copy cleanup.
func (t *Template) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	_, err := t.admin.ExecContext(ctx, dropSQL(t.name))
	if err != nil {
		err = fmt.Errorf("drop template: %w", err)
	}
	return errors.Join(err, t.admin.Close())
}

func createSQL(name string) string {
	return fmt.Sprintf("CREATE DATABASE %s", pgx.Identifier{name}.Sanitize())
}

func cloneSQL(name, template string) string {
	return fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s",
		pgx.Identifier{name}.Sanitize(), pgx.Identifier{template}.Sanitize())
}

func dropSQL(name string) string {
	return fmt.Sprintf(
		"DROP DATABASE IF EXISTS %s WITH (FORCE)", pgx.Identifier{name}.Sanitize())
}

const terminateSQL = `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`

func generateName(prefix string) (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate database name: %w", err)
	}
	return prefix + hex.EncodeToString(b), nil
}
