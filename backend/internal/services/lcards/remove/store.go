package remove

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/apotourlyan/ludustudii/internal/kit"
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
// kit.ErrNotFound for an unknown ID or kit.ErrVersionConflict for a stale version.
func (s *Store) Delete(ctx context.Context, p deleteParams) error {
	query := `WITH del AS (
 DELETE FROM learning_cards WHERE id = $1 AND version = $2 RETURNING id
 ) SELECT EXISTS (SELECT 1 FROM del), EXISTS (SELECT 1 FROM learning_cards WHERE id = $1)`
	var deleted, exists bool
	if err := s.db.QueryRowContext(ctx, query, p.ID, p.Version).Scan(&deleted, &exists); err != nil {
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
