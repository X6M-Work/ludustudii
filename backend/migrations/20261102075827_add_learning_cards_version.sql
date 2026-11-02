-- +goose Up
ALTER TABLE learning_cards ADD COLUMN version bigint NOT NULL DEFAULT 1 CHECK (version >= 1);

-- +goose Down
ALTER TABLE learning_cards DROP COLUMN version;
