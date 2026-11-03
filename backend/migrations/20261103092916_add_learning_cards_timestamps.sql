-- +goose Up
ALTER TABLE learning_cards
    ADD COLUMN created_at timestamptz,
    ADD COLUMN updated_at timestamptz;

UPDATE learning_cards
SET created_at = uuid_extract_timestamp(id),
    updated_at = uuid_extract_timestamp(id);

ALTER TABLE learning_cards
    ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN updated_at SET NOT NULL,
    ADD CONSTRAINT learning_cards_timestamps_check CHECK (updated_at >= created_at);

-- +goose Down
ALTER TABLE learning_cards
    DROP CONSTRAINT learning_cards_timestamps_check,
    DROP COLUMN updated_at,
    DROP COLUMN created_at;
