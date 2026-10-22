-- increment_score relies on ON CONFLICT (username), which needs a unique index.
-- Tables created by the old startup code already have one under this name
-- (from the inline UNIQUE), so this is a no-op for them; tables created by
-- the old create_table endpoint don't.

-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS pokemon_scores_username_key ON pokemon_scores (username);
