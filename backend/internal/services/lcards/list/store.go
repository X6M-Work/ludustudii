package list

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

func (s *Store) List(ctx context.Context) ([]domain.Card, error) {
	query := `SELECT id, title, description FROM learning_cards ORDER BY id DESC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query cards: %w", err)
	}
	defer rows.Close()

	var cards []domain.Card
	for rows.Next() {
		var card domain.Card
		if err := rows.Scan(&card.ID, &card.Title, &card.Description); err != nil {
			return nil, fmt.Errorf("scan card: %w", err)
		}
		cards = append(cards, card)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cards: %w", err)
	}

	return cards, nil
}

