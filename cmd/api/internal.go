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
	r := ginrouter.Default(ctx, "internal")
	r.Use(o11ygin.ClientCancelled())

	a := &API{Router: r}
	o11y.Log(ctx, "New Internal router is called")
	r.GET("/api/private/hello", a.HelloWorldHandler)
	r.GET("/api/private/list_tables", a.ListTablesHandler)
	r.POST("/api/private/create_table", a.CreateTableHandler)
	r.DELETE("/api/private/delete_table", a.DeleteTableHandler)
	r.GET("/api/private/get_answer", a.ReadAnswerFromDBHandler)
	r.GET("/api/private/get_current_score", a.GetScoreHandler)
	r.POST("/api/private/update_user_score", a.UpdateScoreForUserHandler)

	return a, nil
}

func (a *API) Handler() *gin.Engine {
	return a.Router
}
