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

// Create inserts a card and returns its database-generated ID and version.
func (s *Store) Create(ctx context.Context, p createParams) (string, int64, error) {
	query := `INSERT INTO learning_cards (title, description) VALUES ($1, $2) RETURNING id, version`
	var id string
	var version int64
	err := s.db.
		QueryRowContext(ctx, query, p.Title, p.Description).
		Scan(&id, &version)
	return id, version, err
}
