package list

import (
	"context"
	"database/sql"
	"fmt"
)

// Store reads pages of learning cards from the database.
type Store struct {
	db *sql.DB
}

// NewStore returns a Store that uses db.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type listParams struct {
	Limit  int
	Cursor string
}

type row struct {
	ID          string
	Title       string
	Description string
}

type page struct {
	Rows       []row
	NextCursor string
}

// List returns up to p.Limit cards ordered by descending ID, which for UUIDv7
// is newest first, starting after p.Cursor if it is set. NextCursor is the
// last returned ID when more cards follow, and empty otherwise. p.Limit must
// be at least 1.
func (s *Store) List(ctx context.Context, p listParams) (page, error) {
	if p.Limit < 1 {
		return page{}, fmt.Errorf("invalid limit %d < 1", p.Limit)
	}

	// Fetch one extra row to learn whether another page exists.
	query := `SELECT id, title, description FROM learning_cards ORDER BY id DESC LIMIT $1`
	args := []any{p.Limit + 1}
	if p.Cursor != "" {
		query = `SELECT id, title, description FROM learning_cards WHERE id < $1 ORDER BY id DESC LIMIT $2`
		args = []any{p.Cursor, p.Limit + 1}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return page{}, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var data []row
	for rows.Next() {
		var card row
		if err := rows.Scan(&card.ID, &card.Title, &card.Description); err != nil {
			return page{}, fmt.Errorf("scan: %w", err)
		}
		data = append(data, card)
	}
	if err := rows.Err(); err != nil {
		return page{}, fmt.Errorf("iterate: %w", err)
	}

	var next string
	if len(data) > p.Limit {
		data = data[:p.Limit]
		next = data[len(data)-1].ID
	}

	return page{Rows: data, NextCursor: next}, nil
}
