package get

import (
	"context"
	"database/sql"
	"errors"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
)

// Store reads learning cards from the database.
type Store struct {
	db *sql.DB
}

// NewStore returns a Store that uses db.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type row struct {
	ID          string
	Title       string
	Description string
	Version     int64
}

// Get returns the card with the given ID, or domain.ErrNotFound.
func (s *Store) Get(ctx context.Context, id string) (row, error) {
	query := `SELECT id, title, description, version FROM learning_cards WHERE id = $1`
	var data row
	err := s.db.
		QueryRowContext(ctx, query, id).
		Scan(&data.ID, &data.Title, &data.Description, &data.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return data, domain.ErrNotFound
	}
	return data, err
}
