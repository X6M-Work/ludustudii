package update

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
)

type updateParams struct {
	ExpectedVersion int64
	ID              string
	Title           string
	Description     string
}

// Store updates learning cards in the database.
type Store struct {
	db *sql.DB
}

// NewStore returns a Store that uses db.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Update conditionally replaces card p.ID and increments its version atomically.
// It returns the new version, ErrNotFound for an unknown ID, or
// ErrVersionConflict when p.ExpectedVersion is stale.
func (s *Store) Update(ctx context.Context, p updateParams) (int64, error) {
	query := `UPDATE learning_cards SET title = $1, description = $2, version = version + 1 WHERE id = $3 AND version = $4 RETURNING version`
	var version int64
	err := s.db.QueryRowContext(ctx, query, p.Title, p.Description, p.ID, p.ExpectedVersion).Scan(&version)
	if err == nil {
		return version, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("update: %w", err)
	}
	var exists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM learning_cards WHERE id = $1)`, p.ID).Scan(&exists); err != nil {
		return 0, fmt.Errorf("check card exists: %w", err)
	}
	if !exists {
		return 0, domain.ErrNotFound
	}
	return 0, domain.ErrVersionConflict
}
