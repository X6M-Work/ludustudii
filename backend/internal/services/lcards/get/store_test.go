//go:build integration

package get

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
	"github.com/apotourlyan/ludustudii/internal/testkit"
)

func TestMain(m *testing.M) {
	os.Exit(testkit.Run(m))
}

func insertCard(t *testing.T, db *sql.DB, id, title, description string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO learning_cards (id, title, description) VALUES ($1, $2, $3)`,
		id, title, description,
	)
	if err != nil {
		t.Fatalf("insert card: %v", err)
	}
}

func TestStore_Get(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)

	const (
		id1 = "01a1730e-199d-7f9f-b59d-3bcd6f84066a"
		id2 = "01a1730e-199d-7f9f-b59d-3bcd6f84066b"
	)
	insertCard(t, db, id1, "Title 1", "Description 1")
	insertCard(t, db, id2, "Title 2", "Description 2")

	got, err := s.Get(t.Context(), id2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := row{
		ID:          id2,
		Title:       "Title 2",
		Description: "Description 2",
	}
	if got != want {
		t.Errorf("row: got %+v, want %+v", got, want)
	}
}

func TestStore_Get_NotFound(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)

	_, err := s.Get(t.Context(), "01a1730e-199d-7f9f-b59d-3bcd6f84066c")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err: got %v, want %v", err, domain.ErrNotFound)
	}
}

// TestStore_Get_RejectsNonUUID pins that learning_cards.id is a UUID column
// rather than TEXT (which would return domain.ErrNotFound instead of 22P02).
func TestStore_Get_RejectsNonUUID(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)

	_, err := s.Get(t.Context(), "not-a-uuid")
	testkit.AssertPgCode(t, err, testkit.PgCodeInvalidTextRepresentation)
}

func TestStore_Get_CanceledContext(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := s.Get(ctx, "01a1730e-199d-7f9f-b59d-3bcd6f84066c")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err: got %v, want %v", err, context.Canceled)
	}
}
