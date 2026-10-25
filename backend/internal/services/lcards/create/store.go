package create

import (
	"context"
	"database/sql"
)

type createParams struct {
	Title       string
	Description string
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Create(ctx context.Context, p createParams) (string, error) {
	query := `INSERT INTO learning_cards (title, description) VALUES ($1, $2) RETURNING id`
	var id string
	err := s.db.
		QueryRowContext(ctx, query, p.Title, p.Description).
		Scan(&id)
	return id, err
}
