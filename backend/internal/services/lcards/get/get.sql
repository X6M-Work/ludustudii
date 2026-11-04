-- $1 id
SELECT id, title, description, version
FROM learning_cards
WHERE id = $1;
