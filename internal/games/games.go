package games

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/circleci/ex/o11y"
	"github.com/mtslzr/pokeapi-go"
)

func randomNumber() (number int) {
	return rand.Intn(1025) + 1
}

func GetPokemon(ctx context.Context) (string, error) {
	var err error

	ctx, getPokemon := o11y.StartSpan(ctx, "GetPokemon")
	defer o11y.End(getPokemon, &err)

	o11y.AddFieldToTrace(ctx, "before-time", time.Now())

	randomNumber := randomNumber()
	pokemon, err := pokeapi.Resource("pokemon", randomNumber, 1025)
	if err != nil {
		o11y.AddFieldToTrace(ctx, "pokemon-error", err)
		return "", fmt.Errorf("there was an error getting a Pokemon: %s", err)
	}

	o11y.AddFieldToTrace(ctx, "after-time", time.Now())

	return pokemon.Results[0].Name, nil
}
