package update

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Update(ctx context.Context, card domain.Card) error {
	query := `UPDATE learning_cards SET title = $1, description = $2 WHERE id = $3`
	res, err := s.db.ExecContext(ctx, query, card.Title, card.Description, card.ID)
	if err != nil {
		return fmt.Errorf("update card: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrCardNotFound
	}

	return nil
}

