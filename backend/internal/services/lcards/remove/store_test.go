//go:build integration

package remove

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
		`INSERT INTO learning_cards (id, title, description, created_at, updated_at) VALUES ($1, $2, $3, now(), now())`,
		id, title, description,
	)
	if err != nil {
		t.Fatalf("insert card: %v", err)
	}
}

func cardExists(t *testing.T, db *sql.DB, id string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRowContext(t.Context(),
		`SELECT EXISTS (SELECT 1 FROM learning_cards WHERE id = $1)`, id,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("check card exists: %v", err)
	}
	return exists
}

func TestStore_Delete(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)

	const (
		id1 = "01a1730e-199d-7f9f-b59d-3bcd6f84066a"
		id2 = "01a1730e-199d-7f9f-b59d-3bcd6f84066b"
	)
	insertCard(t, db, id1, "Title 1", "Description 1")
	insertCard(t, db, id2, "Title 2", "Description 2")

	if err := s.Delete(t.Context(), deleteParams{ID: id1, Version: 1}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cardExists(t, db, id1) {
		t.Errorf("card %s still exists after delete", id1)
	}
	if !cardExists(t, db, id2) {
		t.Errorf("other card %s was deleted", id2)
	}
}

func TestStore_Delete_NotFound(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)

	err := s.Delete(t.Context(), deleteParams{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066c", Version: 1})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err: got %v, want %v", err, domain.ErrNotFound)
	}
}

// TestStore_Delete_RejectsNonUUID pins that learning_cards.id is a UUID column
// rather than TEXT (which would return domain.ErrNotFound instead of 22P02).
func TestStore_Delete_RejectsNonUUID(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)

	err := s.Delete(t.Context(), deleteParams{ID: "not-a-uuid", Version: 1})
	testkit.AssertPgCode(t, err, testkit.PgCodeInvalidTextRepresentation)
}

func TestStore_Delete_CanceledContext(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := s.Delete(ctx, deleteParams{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066c", Version: 1})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err: got %v, want %v", err, context.Canceled)
	}
}

func TestStore_Delete_StaleVersion(t *testing.T) {
	t.Parallel()
	db := testkit.NewDB(t)
	s := NewStore(db)
	const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
	insertCard(t, db, id, "Original", "Original description")
	if err := s.Delete(t.Context(), deleteParams{ID: id, Version: 99}); !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("stale delete: %v", err)
	}
	var title, description string
	var version int64
	if err := db.QueryRowContext(t.Context(), `SELECT title, description, version FROM learning_cards WHERE id = $1`, id).Scan(&title, &description, &version); err != nil {
		t.Fatal(err)
	}
	if title != "Original" || description != "Original description" || version != 1 {
		t.Fatalf("stale delete changed row: %s, %s, %d", title, description, version)
	}
	if err := s.Delete(t.Context(), deleteParams{ID: id, Version: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(t.Context(), deleteParams{ID: id, Version: 99}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown card: %v", err)
	}
}
