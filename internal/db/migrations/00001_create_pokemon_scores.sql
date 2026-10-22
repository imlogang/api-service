-- pokemon_scores already exists in production, created by the startup code
-- this migration replaces, so every statement here must be a no-op on it.

-- +goose Up
CREATE TABLE IF NOT EXISTS pokemon_scores (
	id SERIAL PRIMARY KEY
);

ALTER TABLE pokemon_scores
	ADD COLUMN IF NOT EXISTS username TEXT UNIQUE,
	ADD COLUMN IF NOT EXISTS score INTEGER NOT NULL DEFAULT 0;
