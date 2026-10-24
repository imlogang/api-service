-- collections records which answers each user has guessed correctly, per
-- game: a Pokédex for the Pokémon game. shiny is true once the user has
-- caught a shiny one.

-- +goose Up
CREATE TABLE collections (
	game TEXT NOT NULL,
	username TEXT NOT NULL,
	item_id INTEGER NOT NULL,
	shiny BOOLEAN NOT NULL DEFAULT false,
	first_caught_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (game, username, item_id)
);
