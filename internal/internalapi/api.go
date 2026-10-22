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

type scoreResponse struct {
	Username string `json:"username"`
	Score    int    `json:"score"`
}

type leaderboardResponse struct {
	Leaderboard []db.LeaderboardEntry `json:"leaderboard"`
}

type pokemonResponse struct {
	Pokemon string `json:"pokemon"`
}

type roundResponse struct {
	Game string `json:"game"`
	games.Round
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

	c.JSON(http.StatusOK, scoreResponse{Username: username, Score: score})
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

	pokemon, err := a.getPokemon(ctx)
	if err != nil {
		o11y.AddFieldToTrace(ctx, "db-error", err)
		c.JSON(http.StatusInternalServerError, returnBody{Error: err.Error()})
		return
	}

	o11y.AddFieldToTrace(ctx, "pokemon", pokemon)
	c.JSON(http.StatusOK, pokemonResponse{Pokemon: pokemon})
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
	if leaderboard == nil {
		leaderboard = []db.LeaderboardEntry{}
	}
	c.JSON(http.StatusOK, leaderboardResponse{Leaderboard: leaderboard})
}

// IncrementScoreHandler adds one point to a user's score atomically and
// returns the new score.
func (a *API) IncrementScoreHandler(c *gin.Context) {
	ctx := c.Request.Context()
	var requestBody requestBody
	err := c.BindJSON(&requestBody)
	ctx, incrementScoreSpan := o11y.StartSpan(ctx, "IncrementScoreHandler")
	defer o11y.End(incrementScoreSpan, &err)

	if err != nil {
		c.JSON(http.StatusBadRequest, returnBody{Error: err.Error()})
		return
	}
	if requestBody.TableName == "" || requestBody.User == "" {
		c.JSON(http.StatusBadRequest, returnBody{Error: "table_name and username required"})
		return
	}

	score, err := a.store.IncrementScore(ctx, requestBody.TableName, requestBody.User)
	if err != nil {
		o11y.AddFieldToTrace(ctx, "db-error", err)
		c.JSON(http.StatusInternalServerError, returnBody{Error: err.Error()})
		return
	}
	o11y.AddFieldToTrace(ctx, "score", score)
	c.JSON(http.StatusOK, scoreResponse{Username: requestBody.User, Score: score})
}

// GetRoundHandler returns one round of the requested game: the answer, the
// options to show (answer included) and an image.
func (a *API) GetRoundHandler(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Query("game")

	var err error
	ctx, getRoundSpan := o11y.StartSpan(ctx, "GetRoundHandler")
	defer o11y.End(getRoundSpan, &err)
	o11y.AddFieldToTrace(ctx, "game", name)

	if name == "" {
		c.JSON(http.StatusBadRequest, returnBody{Error: "game required"})
		return
	}
	game, ok := a.games[name]
	if !ok {
		c.JSON(http.StatusBadRequest, returnBody{Error: fmt.Sprintf("unknown game %q", name)})
		return
	}

	round, err := game.Round(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, returnBody{Error: err.Error()})
		return
	}
	o11y.AddFieldToTrace(ctx, "answer", round.Answer)
	c.JSON(http.StatusOK, roundResponse{Game: name, Round: round})
}
