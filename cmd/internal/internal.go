package httpapi

import (
	"context"
	"github.com/circleci/ex/httpserver/ginrouter"
	"github.com/circleci/ex/o11y"
	"github.com/circleci/ex/o11y/wrappers/o11ygin"
	"github.com/gin-gonic/gin"
)

type API struct {
	Router *gin.Engine
}

func New(ctx context.Context) (*API, error) {
	r := ginrouter.Default(ctx, "internal-api-service")
	r.Use(o11ygin.ClientCancelled())

	a := &API{Router: r}
	o11y.Log(ctx, "New Internal router is called")
	private := r.Group("api/private")
	{
		private.GET("/hello", a.HelloWorldHandler)
		private.GET("/list_tables", a.ListTablesHandler)
		private.POST("/create_table", a.CreateTableHandler)
		private.DELETE("/delete_table", a.DeleteTableHandler)
		private.GET("/get_answer", a.ReadAnswerFromDBHandler)
		private.GET("/get_current_score", a.GetScoreHandler)
		private.POST("/update_user_score", a.UpdateScoreForUserHandler)
		private.GET("/get_pokemon", a.GetPokemonHandler)
		private.GET("/leaderboard", a.LeaderboardHandler)
		private.PUT("/update_table_with_user", a.UpdateTableWithUserHandler)
	}

	return a, nil
}

func (a *API) Handler() *gin.Engine {
	return a.Router
}
