-- +goose Up
CREATE TABLE learning_cards (
    id          UUID PRIMARY KEY DEFAULT uuidv7(),
    title       TEXT NOT NULL CHECK (char_length(title) <= 200),
    description TEXT NOT NULL CHECK (char_length(description) <= 2000)
);

-- +goose Down
DROP TABLE IF EXISTS learning_cards;
