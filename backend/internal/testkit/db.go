// Package testkit provides helpers for tests: isolated PostgreSQL databases
// cloned from a migrated template, and HTTP request and assertion helpers.
package testkit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/apotourlyan/ludustudii/migrations"
	"github.com/jackc/pgx/v5/pgconn"
)

// PostgreSQL SQLSTATE codes for use with AssertPgCode.
const (
	PgCodeCheckViolation            = "23514"
	PgCodeInvalidTextRepresentation = "22P02"
)

var getTemplate func() (*Template, error)

// Run configures a shared migrated template database created lazily on the
// first call to NewDB, runs the test suite, closes the template if one was
// created, and returns the exit code of m.Run. Call it from TestMain:
//
//	func TestMain(m *testing.M) {
//		os.Exit(testkit.Run(m))
//	}
func Run(m *testing.M) int {
	var tmpl *Template
	getTemplate = sync.OnceValues(func() (*Template, error) {
		t, err := NewTemplate(context.Background(), migrations.Up)
		tmpl = t
		return t, err
	})
	defer func() {
		getTemplate = nil
		if tmpl != nil {
			if err := tmpl.Close(); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
	}()
	return m.Run()
}

// NewDB returns a connection to an isolated, migrated database copy that is
// closed and dropped when the test finishes. Run must be called in TestMain,
// and DB_TEST_CONNECTION must be set (see NewTemplate).
func NewDB(t *testing.T) *sql.DB {
	t.Helper()
	if getTemplate == nil {
		t.Fatal("testkit: NewDB called without testkit.Run in TestMain")
	}
	tmpl, err := getTemplate()
	if err != nil {
		t.Fatal(err)
	}
	db, cleanup, err := tmpl.Clone(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := cleanup(); err != nil {
			t.Error(err)
		}
	})
	return db
}

// AssertPgCode fails the test unless err wraps a *pgconn.PgError with the given
// SQLSTATE code, and returns the underlying *pgconn.PgError.
func AssertPgCode(t *testing.T, err error, code string) *pgconn.PgError {
	t.Helper()
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok {
		t.Fatalf("err: got %v, want *pgconn.PgError %s", err, code)
	}
	if pgErr.Code != code {
		t.Fatalf("pg code: got %s (%s), want %s", pgErr.Code, pgErr.Message, code)
	}
	return pgErr
}

// AssertPgConstraint fails the test unless err is a Postgres check_violation
// (23514) for the named constraint.
func AssertPgConstraint(t *testing.T, err error, constraint string) {
	t.Helper()
	pgErr := AssertPgCode(t, err, PgCodeCheckViolation)
	if pgErr.ConstraintName != constraint {
		t.Errorf("pg constraint: got %q, want %q", pgErr.ConstraintName, constraint)
	}
}
