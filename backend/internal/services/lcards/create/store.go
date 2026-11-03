package create

import (
	"context"
	"database/sql"
)

type createParams struct {
	Title       string
	Description string
}

type row struct {
	ID      string
	Version int64
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
func (s *Store) Create(ctx context.Context, p createParams) (row, error) {
	query := `INSERT INTO learning_cards (title, description, created_at, updated_at) VALUES ($1, $2, now(), now()) RETURNING id, version`
	var data row
	err := s.db.
		QueryRowContext(ctx, query, p.Title, p.Description).
		Scan(&data.ID, &data.Version)
	return data, err
}
