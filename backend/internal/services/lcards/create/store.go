package create

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

func (s *Store) Create(ctx context.Context, card domain.Card) (domain.Card, error) {
	query := `INSERT INTO learning_cards (title, description) VALUES ($1, $2) RETURNING id`
	if err := s.db.QueryRowContext(ctx, query, card.Title, card.Description).Scan(&card.ID); err != nil {
		return domain.Card{}, fmt.Errorf("insert card: %w", err)
	}
	return card, nil
}

