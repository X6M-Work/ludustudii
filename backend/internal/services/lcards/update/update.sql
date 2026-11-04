-- $1 title, $2 description, $3 id, $4 expected version
WITH upd AS (
    UPDATE learning_cards
    SET title = $1, description = $2, version = version + 1, updated_at = now()
    WHERE id = $3 AND version = $4
    RETURNING version
)
SELECT (
    SELECT version
    FROM upd
), EXISTS (
    SELECT 1
    FROM learning_cards
    WHERE id = $3
);
