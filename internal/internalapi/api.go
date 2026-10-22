package httpapi

import (
	"errors"
	"fmt"
	"github.com/circleci/ex/o11y"
	"github.com/gin-gonic/gin"
	"github.com/imlogang/api-service/internal/db"
	"github.com/imlogang/api-service/internal/games"
	"net/http"
)

type requestBody struct {
	TableName string `json:"table_name"`
	User      string `json:"username"`
	Score     int    `json:"score"`
	Column    string `json:"column"`
}

type returnBody struct {
	Hello        string `json:"hello,omitempty"`
	UpdateAnswer string `json:"update_answer,omitempty"`
	Error        string `json:"error,omitempty"`
	AddedUser    string `json:"added_user,omitempty"`
}

func (a *API) HelloWorldHandler(c *gin.Context) {
	c.JSON(http.StatusOK, returnBody{Hello: "Hello world!"})
}

func (a *API) UpdateTableWithUserHandler(c *gin.Context) {
	var requestBody requestBody
	ctx := c.Request.Context()
	err := c.BindJSON(&requestBody)

	ctx, updateTableWithUserSpan := o11y.StartSpan(ctx, "UpdateTableWithUserHandler")
	defer o11y.End(updateTableWithUserSpan, &err)
	if err != nil {
		o11y.AddFieldToTrace(ctx, "update-table-with-user", requestBody)
		c.JSON(http.StatusBadRequest, returnBody{Error: err.Error()})
		return
	}

	_, err = a.store.UpdateTableWithUser(requestBody.TableName, requestBody.User, ctx)
	if err != nil {
		o11y.AddFieldToTrace(ctx, "db-error", err)
		c.JSON(http.StatusInternalServerError, returnBody{Error: err.Error()})
		return
	}
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, returnBody{AddedUser: requestBody.User})
}

func (a *API) GetScoreHandler(c *gin.Context) {
	ctx := c.Request.Context()

	tableName := c.Query("tablename")
	username := c.Query("username")

	var err error
	ctx, getScoreHandlerSpan := o11y.StartSpan(ctx, "GetScoreHandler")
	defer o11y.End(getScoreHandlerSpan, &err)
	if tableName == "" || username == "" {
		o11y.AddFieldToTrace(ctx, "table_name", tableName)
		o11y.AddFieldToTrace(ctx, "username", username)
		c.JSON(http.StatusBadRequest, returnBody{Error: "tablename or username required"})
		return
	}

	score, err := a.store.GetCurrentScore(tableName, username, ctx)
	if err != nil {
		o11y.AddFieldToTrace(ctx, "db-error", err)
		c.JSON(http.StatusInternalServerError, returnBody{Error: err.Error()})
		return
	}

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, "Score for %s: %d\n", username, score)
}

func (a *API) UpdateScoreForUserHandler(c *gin.Context) {
	ctx := c.Request.Context()
	var requestBody requestBody
	err := c.BindJSON(&requestBody)
	ctx, updateScoreForUserHandler := o11y.StartSpan(ctx, "UpdateScoreForUserHandler")
	defer o11y.End(updateScoreForUserHandler, &err)

	if err != nil {
		o11y.AddFieldToTrace(ctx, "update-score-for-user", requestBody)
		c.JSON(http.StatusBadRequest, returnBody{Error: err.Error()})
		return
	}

	sql, err := a.store.UpdateScoreForUser(requestBody.TableName, requestBody.User, requestBody.Score, requestBody.Column, ctx)
	if errors.Is(err, db.ErrColumnNotUpdatable) {
		c.JSON(http.StatusBadRequest, returnBody{Error: err.Error()})
		return
	}
	if err != nil {
		o11y.AddFieldToTrace(ctx, "db-error", err)
		c.JSON(http.StatusInternalServerError, returnBody{Error: err.Error()})
		return
	}
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, returnBody{UpdateAnswer: sql})
}

func (a *API) GetPokemonHandler(c *gin.Context) {
	ctx := c.Request.Context()
	var err error

	ctx, getPokemonHandlerSpan := o11y.StartSpan(ctx, "GetPokemonHandler")
	defer o11y.End(getPokemonHandlerSpan, &err)

	pokemon, err := games.GetPokemon(ctx)
	if err != nil {
		o11y.AddFieldToTrace(ctx, "db-error", err)
		c.JSON(http.StatusInternalServerError, returnBody{Error: err.Error()})
		return
	}

	o11y.AddFieldToTrace(ctx, "pokemon", pokemon)
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, fmt.Sprintf("%s\n", pokemon))
}

func (a *API) LeaderboardHandler(c *gin.Context) {
	ctx := c.Request.Context()
	tableName := c.Query("tablename")

	var err error
	ctx, leaderboardHandlerSpan := o11y.StartSpan(ctx, "LeaderboardHandler")
	defer o11y.End(leaderboardHandlerSpan, &err)

	if tableName == "" {
		o11y.AddFieldToTrace(ctx, "table-name", tableName)
		c.JSON(http.StatusBadRequest, returnBody{Error: "tablename required"})
		return
	}

	leaderboard, err := a.store.GetLeaderboard(tableName, ctx)
	if err != nil {
		o11y.AddFieldToTrace(ctx, "db-error", err)
		c.JSON(http.StatusInternalServerError, returnBody{Error: err.Error()})
		return
	}
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, fmt.Sprintf("\n%s", leaderboard))
}
