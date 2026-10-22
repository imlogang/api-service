package httpapi

import (
	"context"
	"github.com/circleci/ex/httpserver/ginrouter"
	"github.com/circleci/ex/o11y"
	"github.com/circleci/ex/o11y/wrappers/o11ygin"
	"github.com/gin-gonic/gin"
	"github.com/imlogang/api-service/internal/db"
	"github.com/imlogang/api-service/internal/games"
)

type API struct {
	Router *gin.Engine
	store  *db.Store
	// getPokemon is games.GetPokemon, swappable so tests don't call PokeAPI.
	getPokemon func(ctx context.Context) (string, error)
}

func New(ctx context.Context, store *db.Store) (*API, error) {
	r := ginrouter.Default(ctx, "internal")
	r.Use(o11ygin.ClientCancelled())

	a := &API{Router: r, store: store, getPokemon: games.GetPokemon}
	o11y.Log(ctx, "New Internal router is called")
	r.GET("/api/private/hello", a.HelloWorldHandler)
	r.GET("/api/private/get_current_score", a.GetScoreHandler)
	r.POST("/api/private/update_user_score", a.UpdateScoreForUserHandler)
	r.POST("/api/private/increment_score", a.IncrementScoreHandler)
	r.GET("/api/private/get_pokemon", a.GetPokemonHandler)
	r.GET("/api/private/leaderboard", a.LeaderboardHandler)
	r.PUT("/api/private/update_table_with_user", a.UpdateTableWithUserHandler)

	return a, nil
}

func (a *API) Handler() *gin.Engine {
	return a.Router
}
