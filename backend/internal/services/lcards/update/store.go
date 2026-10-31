package update

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
)

type updateParams struct {
	ID          string
	Title       string
	Description string
}

// Store updates learning cards in the database.
type Store struct {
	db *sql.DB
}

// NewStore returns a Store that uses db.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Update sets the title and description of card p.ID, or returns
// domain.ErrNotFound.
func (s *Store) Update(ctx context.Context, p updateParams) error {
	query := `UPDATE learning_cards SET title = $1, description = $2 WHERE id = $3`
	res, err := s.db.ExecContext(ctx, query, p.Title, p.Description, p.ID)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}
