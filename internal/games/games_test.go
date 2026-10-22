package games

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/circleci/ex/testing/testcontext"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

type fakePokeAPI struct {
	*httptest.Server
	requests atomic.Int32
	fail     atomic.Bool
}

// newFakePokeAPI serves a species list with the given names, numbered from 1.
func newFakePokeAPI(t *testing.T, names ...string) *fakePokeAPI {
	t.Helper()
	f := &fakePokeAPI{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.requests.Add(1)
		if r.URL.Path != "/pokemon-species" {
			http.NotFound(w, r)
			return
		}
		if f.fail.Load() {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		type result struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		}
		var body struct {
			Results []result `json:"results"`
		}
		for i, name := range names {
			body.Results = append(body.Results, result{
				Name: name,
				URL:  fmt.Sprintf("https://pokeapi.co/api/v2/pokemon-species/%d/", i+1),
			})
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(f.Close)
	return f
}

func TestPokemon_Round(t *testing.T) {
	ctx := testcontext.Background()
	names := []string{"bulbasaur", "ivysaur", "venusaur", "charmander", "charmeleon", "charizard"}
	p := NewPokemon(newFakePokeAPI(t, names...).URL)

	for range 50 {
		round, err := p.Round(ctx)
		assert.NilError(t, err)

		assert.Check(t, cmp.Len(round.Options, OptionCount))
		assert.Check(t, cmp.Contains(round.Options, round.Answer))
		sorted := slices.Sorted(slices.Values(round.Options))
		assert.Check(t, cmp.Len(slices.Compact(sorted), OptionCount), "options repeat: %v", round.Options)
		for _, o := range round.Options {
			assert.Check(t, cmp.Contains(names, o))
		}

		id := slices.Index(names, round.Answer) + 1
		assert.Check(t, cmp.Equal(round.Image,
			fmt.Sprintf("https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/%d.png", id)))
	}
}

// Every species, including the first, should be able to come up.
func TestPokemon_RoundCoversEverySpecies(t *testing.T) {
	ctx := testcontext.Background()
	names := []string{"bulbasaur", "ivysaur", "venusaur", "charmander", "charmeleon"}
	p := NewPokemon(newFakePokeAPI(t, names...).URL)

	seen := map[string]bool{}
	for range 500 {
		round, err := p.Round(ctx)
		assert.NilError(t, err)
		seen[round.Answer] = true
	}
	assert.Check(t, cmp.Len(seen, len(names)))
}

func TestPokemon_FetchesSpeciesOnce(t *testing.T) {
	ctx := testcontext.Background()
	api := newFakePokeAPI(t, "bulbasaur", "ivysaur", "venusaur", "charmander")
	p := NewPokemon(api.URL + "/")

	for range 10 {
		_, err := p.Round(ctx)
		assert.NilError(t, err)
	}
	assert.Check(t, cmp.Equal(api.requests.Load(), int32(1)))
}

func TestPokemon_RetriesAfterFailedFetch(t *testing.T) {
	ctx := testcontext.Background()
	api := newFakePokeAPI(t, "bulbasaur", "ivysaur", "venusaur", "charmander")
	api.fail.Store(true)
	p := NewPokemon(api.URL)

	_, err := p.Round(ctx)
	assert.Check(t, cmp.ErrorContains(err, "503"))

	api.fail.Store(false)
	_, err = p.Round(ctx)
	assert.NilError(t, err)
	assert.Check(t, cmp.Equal(api.requests.Load(), int32(2)))
}

func TestPokemon_TooFewSpecies(t *testing.T) {
	p := NewPokemon(newFakePokeAPI(t, "bulbasaur", "ivysaur", "venusaur").URL)

	_, err := p.Round(testcontext.Background())
	assert.Check(t, cmp.ErrorContains(err, "need at least 4 species"))
}
