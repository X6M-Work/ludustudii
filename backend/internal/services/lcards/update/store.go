package update

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	"github.com/apotourlyan/ludustudii/internal/kit"
)

//go:embed update.sql
var updateSQL string

type updateParams struct {
	Version     int64
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

// Update conditionally replaces card p.ID and increments its version atomically.
// It returns the new version, ErrNotFound for an unknown ID, or
// ErrVersionConflict when p.Version is stale.
func (s *Store) Update(ctx context.Context, p updateParams) (int64, error) {
	var version sql.NullInt64
	var exists bool
	if err := s.db.QueryRowContext(ctx, updateSQL, p.Title, p.Description, p.ID, p.Version).Scan(&version, &exists); err != nil {
		return 0, fmt.Errorf("update: %w", err)
	}
	if version.Valid {
		return version.Int64, nil
	}
	if !exists {
		return 0, kit.ErrNotFound
	}
	return 0, kit.ErrVersionConflict
}
