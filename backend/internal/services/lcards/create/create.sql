-- $1 title, $2 description
INSERT INTO learning_cards (title, description, created_at, updated_at)
VALUES ($1, $2, now(), now())
RETURNING id, version;
