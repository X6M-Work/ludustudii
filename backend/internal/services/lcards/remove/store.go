package remove

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
)

// Store deletes learning cards from the database.
type Store struct {
	db *sql.DB
}

// NewStore returns a Store that uses db.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Delete removes the card only when expectedVersion matches. It returns
// domain.ErrNotFound for an unknown ID or domain.ErrVersionConflict for a stale version.
func (s *Store) Delete(ctx context.Context, id string, expectedVersion int64) error {
	query := `DELETE FROM learning_cards WHERE id = $1 AND version = $2`
	res, err := s.db.ExecContext(ctx, query, id, expectedVersion)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		var exists bool
		if err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM learning_cards WHERE id = $1)`, id).Scan(&exists); err != nil {
			return fmt.Errorf("check card exists: %w", err)
		}
		if !exists {
			return domain.ErrNotFound
		}
		return domain.ErrVersionConflict
	}

	return nil
}
