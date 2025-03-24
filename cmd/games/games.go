package games

import (
	"fmt"
	"math/rand"
	"github.com/mtslzr/pokeapi-go"
)

func randomNumber() (number int) {
	return rand.Intn(1025) + 1
}

func GetPokemon() (string, error) {
	randomNumber := randomNumber()
	pokemon, err := pokeapi.Resource("pokemon", randomNumber, 0)
	if err != nil {
		return "", fmt.Errorf("there was an error: %s", err)
	}
	return pokemon.Results[0].Name, nil
}
