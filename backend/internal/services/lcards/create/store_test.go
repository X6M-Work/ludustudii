//go:build integration

package create

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/apotourlyan/ludustudii/internal/kit/validator"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
	"github.com/apotourlyan/ludustudii/internal/testkit"
)

func TestMain(m *testing.M) {
	os.Exit(testkit.Run(m))
}

func TestStore_Create(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)

	want := createParams{
		Title:       "Caf\u00E9 Title",
		Description: "Caf\u00E9 Description",
	}

	id, err := s.Create(t.Context(), want)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validator.UUID7("id", id); err != nil {
		t.Errorf("id %q: %v", id, err)
	}

	var got createParams
	err = db.QueryRowContext(t.Context(),
		`SELECT title, description FROM learning_cards WHERE id = $1`, id,
	).Scan(&got.Title, &got.Description)
	if err != nil {
		t.Fatalf("query created row: %v", err)
	}
	if got != want {
		t.Errorf("row: got %+v, want %+v", got, want)
	}
}

func TestStore_Create_MaxLengths(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)

	want := createParams{
		Title:       strings.Repeat("\u00E9", domain.TitleMaxLength),
		Description: strings.Repeat("\u00E9", domain.DescriptionMaxLength),
	}

	id, err := s.Create(t.Context(), want)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got createParams
	err = db.QueryRowContext(t.Context(),
		`SELECT title, description FROM learning_cards WHERE id = $1`, id,
	).Scan(&got.Title, &got.Description)
	if err != nil {
		t.Fatalf("query created row: %v", err)
	}
	if got != want {
		t.Errorf("row lengths: got (%d, %d), want (%d, %d)",
			utf8.RuneCountInString(got.Title), utf8.RuneCountInString(got.Description),
			utf8.RuneCountInString(want.Title), utf8.RuneCountInString(want.Description))
	}
}

func TestStore_Create_CheckConstraints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		params         createParams
		wantConstraint string
	}{
		{
			name: "empty title",
			params: createParams{
				Title:       "",
				Description: "Description",
			},
			wantConstraint: "learning_cards_title_check",
		},
		{
			name: "title over max length",
			params: createParams{
				Title:       strings.Repeat("a", domain.TitleMaxLength+1),
				Description: "Description",
			},
			wantConstraint: "learning_cards_title_check",
		},
		{
			name: "empty description",
			params: createParams{
				Title:       "Title",
				Description: "",
			},
			wantConstraint: "learning_cards_description_check",
		},
		{
			name: "description over max length",
			params: createParams{
				Title:       "Title",
				Description: strings.Repeat("a", domain.DescriptionMaxLength+1),
			},
			wantConstraint: "learning_cards_description_check",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := testkit.NewDB(t)
			s := NewStore(db)

			_, err := s.Create(t.Context(), tt.params)
			testkit.AssertPgConstraint(t, err, tt.wantConstraint)
		})
	}
}

func TestStore_Create_CanceledContext(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	s := NewStore(db)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := s.Create(ctx, createParams{Title: "Title", Description: "Description"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err: got %v, want %v", err, context.Canceled)
	}
}
