-- $1 limit plus one
SELECT id, title, description, version
FROM learning_cards
ORDER BY id DESC
LIMIT $1;
