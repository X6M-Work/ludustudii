package remove

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
)

type deleteParams struct {
	ID      string
	Version int64
}

// Store deletes learning cards from the database.
type Store struct {
	db *sql.DB
}

// NewStore returns a Store that uses db.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Delete removes the card only when p.Version matches. It returns
// domain.ErrNotFound for an unknown ID or domain.ErrVersionConflict for a stale version.
func (s *Store) Delete(ctx context.Context, p deleteParams) error {
	query := `DELETE FROM learning_cards WHERE id = $1 AND version = $2`
	res, err := s.db.ExecContext(ctx, query, p.ID, p.Version)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		var exists bool
		if err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM learning_cards WHERE id = $1)`, p.ID).Scan(&exists); err != nil {
			return fmt.Errorf("check card exists: %w", err)
		}
		if !exists {
			return domain.ErrNotFound
		}
		return domain.ErrVersionConflict
	}

	return nil
}
