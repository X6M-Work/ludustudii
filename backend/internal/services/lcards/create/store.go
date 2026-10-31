package create

import (
	"context"
	"database/sql"
)

type createParams struct {
	Title       string
	Description string
}

// Store inserts learning cards into the database.
type Store struct {
	db *sql.DB
}

// NewStore returns a Store that uses db.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Create inserts a card and returns the ID the database generated for it.
func (s *Store) Create(ctx context.Context, p createParams) (string, error) {
	query := `INSERT INTO learning_cards (title, description) VALUES ($1, $2) RETURNING id`
	var id string
	err := s.db.
		QueryRowContext(ctx, query, p.Title, p.Description).
		Scan(&id)
	return id, err
}
