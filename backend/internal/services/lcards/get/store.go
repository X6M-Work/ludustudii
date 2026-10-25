package get

import (
	"context"
	"database/sql"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type row struct {
	ID          string
	Title       string
	Description string
}

func (s *Store) Get(ctx context.Context, id string) (row, error) {
	query := `SELECT id, title, description FROM learning_cards WHERE id = $1`
	var data row
	err := s.db.
		QueryRowContext(ctx, query, id).
		Scan(&data.ID, &data.Title, &data.Description)
	return data, err
}
