//go:build integration

package list

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"reflect"
	"testing"

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

func TestStore_List_Empty(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)

	got, err := s.List(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("rows: got %+v, want empty", got)
	}
}

func TestStore_List_OrderedByIDDesc(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)

	cardA := row{
		ID:          "01a1730e-199d-7f9f-b59d-3bcd6f84066a",
		Title:       "Title A",
		Description: "Description A",
	}
	cardB := row{
		ID:          "01a1730e-199d-7f9f-b59d-3bcd6f84066b",
		Title:       "Title B",
		Description: "Description B",
	}
	cardC := row{
		ID:          "01a1730e-199d-7f9f-b59d-3bcd6f84066c",
		Title:       "Title C",
		Description: "Description C",
	}

	// Insert out of order to verify ORDER BY id DESC.
	insertCard(t, db, cardB.ID, cardB.Title, cardB.Description)
	insertCard(t, db, cardA.ID, cardA.Title, cardA.Description)
	insertCard(t, db, cardC.ID, cardC.Title, cardC.Description)

	got, err := s.List(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []row{cardC, cardB, cardA}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("rows: got %+v, want %+v", got, want)
	}
}

func TestStore_List_CanceledContext(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := s.List(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err: got %v, want %v", err, context.Canceled)
	}
}
