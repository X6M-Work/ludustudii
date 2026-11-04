package remove

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	"github.com/apotourlyan/ludustudii/internal/kit"
)

//go:embed remove.sql
var removeSQL string

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
// kit.ErrNotFound for an unknown ID or kit.ErrVersionConflict for a stale version.
func (s *Store) Delete(ctx context.Context, p deleteParams) error {
	var deleted, exists bool
	if err := s.db.QueryRowContext(ctx, removeSQL, p.ID, p.Version).Scan(&deleted, &exists); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if deleted {
		return nil
	}
	if !exists {
		return kit.ErrNotFound
	}
	return kit.ErrVersionConflict
}
