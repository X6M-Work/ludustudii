-- $1 id, $2 expected version
WITH del AS (
    DELETE FROM learning_cards
    WHERE id = $1 AND version = $2
    RETURNING id
)
SELECT EXISTS (
    SELECT 1
    FROM del
) AS deleted, EXISTS (
    SELECT 1
    FROM learning_cards
    WHERE id = $1
) AS card_exists;
