-- +goose Up
CREATE TABLE learning_cards (
    id          UUID PRIMARY KEY DEFAULT uuidv7(),
    title       TEXT NOT NULL CHECK (char_length(title) BETWEEN 1 AND 200),
    description TEXT NOT NULL CHECK (char_length(description) BETWEEN 1 AND 2000),
    version     bigint NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at  timestamptz NOT NULL,
    updated_at  timestamptz NOT NULL,
    CONSTRAINT learning_cards_timestamps_check CHECK (updated_at >= created_at)
);

-- +goose Down
DROP TABLE IF EXISTS learning_cards;
