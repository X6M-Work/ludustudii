package list

import (
	"context"
	"database/sql"
	"fmt"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type row struct {
	ID          string
	Title       string
	Description string
}

func (s *Store) List(ctx context.Context) ([]row, error) {
	query := `SELECT id, title, description FROM learning_cards ORDER BY id DESC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var data []row
	for rows.Next() {
		var card row
		if err := rows.Scan(&card.ID, &card.Title, &card.Description); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		data = append(data, card)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate: %w", err)
	}

	return data, nil
}
