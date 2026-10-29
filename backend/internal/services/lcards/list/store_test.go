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

var (
	cardA = row{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066a", Title: "Title A", Description: "Description A"}
	cardB = row{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066b", Title: "Title B", Description: "Description B"}
	cardC = row{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066c", Title: "Title C", Description: "Description C"}
)

// seedABC inserts the cards out of order to verify ORDER BY id DESC.
func seedABC(t *testing.T, db *sql.DB) {
	t.Helper()
	insertCard(t, db, cardB.ID, cardB.Title, cardB.Description)
	insertCard(t, db, cardA.ID, cardA.Title, cardA.Description)
	insertCard(t, db, cardC.ID, cardC.Title, cardC.Description)
}

func TestStore_List_Empty(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)

	got, err := s.List(t.Context(), listParams{Limit: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Rows) != 0 || got.NextCursor != "" {
		t.Errorf("page: got %+v, want empty", got)
	}
}

func TestStore_List_OrderedByIDDesc(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)
	seedABC(t, db)

	got, err := s.List(t.Context(), listParams{Limit: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := page{Rows: []row{cardC, cardB, cardA}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("page: got %+v, want %+v", got, want)
	}
}

func TestStore_List_Pages(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)
	seedABC(t, db)

	first, err := s.List(t.Context(), listParams{Limit: 2})
	if err != nil {
		t.Fatalf("first page: unexpected error: %v", err)
	}
	wantFirst := page{Rows: []row{cardC, cardB}, NextCursor: cardB.ID}
	if !reflect.DeepEqual(first, wantFirst) {
		t.Errorf("first page: got %+v, want %+v", first, wantFirst)
	}

	second, err := s.List(t.Context(), listParams{Limit: 2, Cursor: first.NextCursor})
	if err != nil {
		t.Fatalf("second page: unexpected error: %v", err)
	}
	wantSecond := page{Rows: []row{cardA}}
	if !reflect.DeepEqual(second, wantSecond) {
		t.Errorf("second page: got %+v, want %+v", second, wantSecond)
	}
}

// Exactly Limit rows left means no further page, which proves the Limit+1 check.
func TestStore_List_ExactlyLimit(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)
	seedABC(t, db)

	got, err := s.List(t.Context(), listParams{Limit: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := page{Rows: []row{cardC, cardB, cardA}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("page: got %+v, want %+v", got, want)
	}
}

func TestStore_List_DeletedCursor(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)
	seedABC(t, db)

	if _, err := db.ExecContext(t.Context(),
		`DELETE FROM learning_cards WHERE id = $1`, cardB.ID,
	); err != nil {
		t.Fatalf("delete card: %v", err)
	}

	got, err := s.List(t.Context(), listParams{Limit: 20, Cursor: cardB.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := page{Rows: []row{cardA}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("page: got %+v, want %+v", got, want)
	}
}

// Without the guard, 0 indexes data[-1] and -1 slices data[:-1]; both panic.
func TestStore_List_InvalidLimit(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)
	seedABC(t, db)

	for _, limit := range []int{0, -1} {
		got, err := s.List(t.Context(), listParams{Limit: limit})
		if err == nil {
			t.Errorf("limit %d: got %v, want error", limit, err)
		}
		if !reflect.DeepEqual(got, page{}) {
			t.Errorf("limit %d: got %+v, want %+v", limit, got, page{})
		}
	}
}

func TestStore_List_CanceledContext(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := s.List(ctx, listParams{Limit: 20})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err: got %v, want %v", err, context.Canceled)
	}
}
