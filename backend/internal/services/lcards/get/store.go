package get

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Get(ctx context.Context, id string) (domain.Card, error) {
	query := `SELECT id, title, description FROM learning_cards WHERE id = $1`
	var card domain.Card
	err := s.db.QueryRowContext(ctx, query, id).Scan(&card.ID, &card.Title, &card.Description)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Card{}, domain.ErrCardNotFound
		}
		return domain.Card{}, fmt.Errorf("query card: %w", err)
	}
	return card, nil
}

