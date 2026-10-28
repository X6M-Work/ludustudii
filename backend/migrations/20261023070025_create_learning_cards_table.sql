-- +goose Up
CREATE TABLE learning_cards (
    id          UUID PRIMARY KEY DEFAULT uuidv7(),
    title       TEXT NOT NULL CHECK (char_length(title) BETWEEN 1 AND 200),
    description TEXT NOT NULL CHECK (char_length(description) BETWEEN 1 AND 2000)
);

-- +goose Down
DROP TABLE IF EXISTS learning_cards;
