package games

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/circleci/ex/o11y"
)

// OptionCount is how many choices a round offers, the answer included.
const OptionCount = 4

// Round is one question of a guessing game. Options holds the answer and
// OptionCount-1 wrong choices, all different, in random order.
type Round struct {
	Answer  string   `json:"answer"`
	Options []string `json:"options"`
	Image   string   `json:"image"`
}

// Game is a guessing game the API can serve rounds for.
type Game interface {
	Round(ctx context.Context) (Round, error)
}

const (
	DefaultPokeAPIURL = "https://pokeapi.co/api/v2"
	spriteURL         = "https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/%d.png"
)

type species struct {
	name string
	id   int
}

// Pokemon is the "guess that Pokémon" game. It fetches the list of every
// species from PokeAPI once and picks rounds from memory after that.
type Pokemon struct {
	baseURL string
	client  *http.Client

	mu      sync.Mutex
	species []species
}

func NewPokemon(baseURL string) *Pokemon {
	return &Pokemon{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *Pokemon) Round(ctx context.Context) (Round, error) {
	all, err := p.allSpecies(ctx)
	if err != nil {
		return Round{}, err
	}
	if len(all) < OptionCount {
		return Round{}, fmt.Errorf("need at least %d species, PokeAPI returned %d", OptionCount, len(all))
	}

	picks := rand.Perm(len(all))[:OptionCount]
	answer := all[picks[0]]
	options := make([]string, OptionCount)
	for i, n := range picks {
		options[i] = all[n].name
	}
	rand.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })

	return Round{
		Answer:  answer.name,
		Options: options,
		Image:   fmt.Sprintf(spriteURL, answer.id),
	}, nil
}

// Random returns the name of one random species.
func (p *Pokemon) Random(ctx context.Context) (string, error) {
	all, err := p.allSpecies(ctx)
	if err != nil {
		return "", err
	}
	if len(all) == 0 {
		return "", errors.New("PokeAPI returned no species")
	}
	return all[rand.IntN(len(all))].name, nil
}

// allSpecies returns the cached species list, fetching it on first use. A
// failed fetch isn't cached, so the next call tries again.
func (p *Pokemon) allSpecies(ctx context.Context) ([]species, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.species != nil {
		return p.species, nil
	}

	all, err := p.fetchSpecies(ctx)
	if err != nil {
		return nil, err
	}
	p.species = all
	return all, nil
}

func (p *Pokemon) fetchSpecies(ctx context.Context) (all []species, err error) {
	ctx, span := o11y.StartSpan(ctx, "games: fetch pokemon species")
	defer o11y.End(span, &err)

	// The limit is above the number of species so new generations show up
	// without a code change.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/pokemon-species?limit=10000", nil)
	if err != nil {
		return nil, err
	}
	res, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("there was an error getting a Pokemon: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("there was an error getting a Pokemon: PokeAPI returned %s", res.Status)
	}

	var body struct {
		Results []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"results"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decoding PokeAPI species list: %w", err)
	}

	all = make([]species, 0, len(body.Results))
	for _, r := range body.Results {
		// URLs look like https://pokeapi.co/api/v2/pokemon-species/25/
		id, err := strconv.Atoi(path.Base(strings.TrimSuffix(r.URL, "/")))
		if err != nil {
			return nil, fmt.Errorf("unexpected species URL %q", r.URL)
		}
		all = append(all, species{name: r.Name, id: id})
	}
	o11y.AddFieldToTrace(ctx, "species_count", len(all))
	return all, nil
}
