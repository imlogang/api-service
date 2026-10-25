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
	// games are the games get_round serves, by name. Tests swap them out so
	// they don't call PokeAPI.
	games map[string]games.Game
}

func New(ctx context.Context, store *db.Store) (*API, error) {
	r := ginrouter.Default(ctx, "internal")
	r.Use(o11ygin.ClientCancelled())

	a := &API{
		Router: r,
		store:  store,
		games:  map[string]games.Game{"pokemon": games.NewPokemon(games.DefaultPokeAPIURL)},
	}
	o11y.Log(ctx, "New Internal router is called")
	r.GET("/api/private/hello", a.HelloWorldHandler)
	r.GET("/api/private/get_current_score", a.GetScoreHandler)
	r.POST("/api/private/update_user_score", a.UpdateScoreForUserHandler)
	r.POST("/api/private/increment_score", a.IncrementScoreHandler)
	r.GET("/api/private/get_round", a.GetRoundHandler)
	r.GET("/api/private/leaderboard", a.LeaderboardHandler)
	r.POST("/api/private/add_to_collection", a.AddToCollectionHandler)
	r.GET("/api/private/get_collection", a.GetCollectionHandler)

	return a, nil
}

func (a *API) Handler() *gin.Engine {
	return a.Router
}
