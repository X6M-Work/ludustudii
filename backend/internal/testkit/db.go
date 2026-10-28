package testkit

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/apotourlyan/ludustudii/migrations"
)

var getTemplate func() (*Template, error)

// Run configures a shared migrated template database created lazily on the
// first call to NewDB, runs the test suite, and closes the template if one was
// created. Call it from TestMain:
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
// closed and dropped when the test finishes. Run must be called in TestMain.
func NewDB(t *testing.T) *sql.DB {
	t.Helper()
	if getTemplate == nil {
		t.Fatal("dbtesting: NewDB called without dbtesting.Run in TestMain")
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
