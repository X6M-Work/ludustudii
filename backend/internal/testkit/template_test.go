//go:build integration

package testkit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func migrate(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE items (id int PRIMARY KEY)`)
	return err
}

// newTemplate creates a template that is closed when the test ends.
// t.Cleanup runs in reverse order, so clone cleanups registered later run first.
func newTemplate(t *testing.T) *Template {
	t.Helper()
	tmpl, err := NewTemplate(t.Context(), migrate)
	if err != nil {
		t.Fatalf("new template: %v", err)
	}
	t.Cleanup(func() {
		if err := tmpl.Close(); err != nil {
			t.Error(err)
		}
	})
	return tmpl
}

func clone(t *testing.T, tmpl *Template) *sql.DB {
	t.Helper()
	db, cleanup, err := tmpl.Clone(t.Context())
	if err != nil {
		t.Fatalf("clone: %v", err)
	}
	t.Cleanup(func() {
		if err := cleanup(); err != nil {
			t.Error(err)
		}
	})
	return db
}

func databaseExists(t *testing.T, name string) bool {
	t.Helper()
	cfg, err := pgx.ParseConfig(os.Getenv(dbConnectionStringKey))
	if err != nil {
		t.Fatal(err)
	}
	db := stdlib.OpenDB(*cfg)
	defer db.Close()

	var exists bool
	err = db.QueryRowContext(t.Context(),
		`SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, name).Scan(&exists)
	if err != nil {
		t.Fatal(err)
	}
	return exists
}

func currentDatabase(t *testing.T, db *sql.DB) string {
	t.Helper()
	var name string
	if err := db.QueryRowContext(t.Context(), `SELECT current_database()`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	return name
}

func countItems(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRowContext(t.Context(), `SELECT count(*) FROM items`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestClone_HasMigratedSchema(t *testing.T) {
	t.Parallel()
	db := clone(t, newTemplate(t))

	if _, err := db.ExecContext(t.Context(), `INSERT INTO items (id) VALUES (1)`); err != nil {
		t.Fatalf("insert into migrated table: %v", err)
	}
}

func TestClone_Isolated(t *testing.T) {
	t.Parallel()
	tmpl := newTemplate(t)
	a, b := clone(t, tmpl), clone(t, tmpl)

	if _, err := a.ExecContext(t.Context(), `INSERT INTO items (id) VALUES (1)`); err != nil {
		t.Fatal(err)
	}
	if n := countItems(t, a); n != 1 {
		t.Errorf("clone a: got %d rows, want 1", n)
	}
	if n := countItems(t, b); n != 0 {
		t.Errorf("clone b: got %d rows, want 0", n)
	}
}

func TestClone_CleanupDropsDatabase(t *testing.T) {
	t.Parallel()
	tmpl := newTemplate(t)

	db, cleanup, err := tmpl.Clone(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	name := currentDatabase(t, db)
	if !databaseExists(t, name) {
		t.Fatalf("%s: does not exist before cleanup", name)
	}

	if err := cleanup(); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if databaseExists(t, name) {
		t.Errorf("%s: still exists after cleanup", name)
	}
}

func TestClose_DropsTemplate(t *testing.T) {
	t.Parallel()
	tmpl, err := NewTemplate(t.Context(), migrate) // not newTemplate: this test closes it itself
	if err != nil {
		t.Fatal(err)
	}
	if !databaseExists(t, tmpl.name) {
		t.Fatalf("%s: does not exist before close", tmpl.name)
	}

	if err := tmpl.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if databaseExists(t, tmpl.name) {
		t.Errorf("%s: still exists after close", tmpl.name)
	}
}

func TestClone_Concurrent(t *testing.T) {
	t.Parallel()
	tmpl := newTemplate(t)

	const n = 10
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for range n {
		wg.Go(func() {
			db, cleanup, err := tmpl.Clone(t.Context())
			if err != nil {
				errs <- err
				return
			}
			name := currentDatabase(t, db)
			if !databaseExists(t, name) {
				errs <- errors.Join(fmt.Errorf("%s: does not exist", name), cleanup())
				return
			}
			errs <- cleanup()
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}
}
