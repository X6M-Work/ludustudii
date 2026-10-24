package delete

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

func (s *Store) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM learning_cards WHERE id = $1`
	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete card: %w", err)
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

