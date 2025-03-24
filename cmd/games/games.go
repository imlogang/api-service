package games

import (
	"fmt"
	"github.com/mtslzr/pokeapi-go"
)

func GetPokemon() (string, error) {
	pokemon, err := pokeapi.Resource("pokemon", 0, 0)
	if err != nil {
		return "", fmt.Errorf("there was an error: %s", err)
	}
	return pokemon.Results[0].Name, nil
}
