package httpapi

import (
	"context"
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
	// Score is a pointer so that leaving it out is an error, not a reset to 0.
	Score  *int   `json:"score"`
	Column string `json:"column"`
}

type returnBody struct {
	Hello        string `json:"hello,omitempty"`
	UpdateAnswer string `json:"update_answer,omitempty"`
	Error        string `json:"error,omitempty"`
}

type scoreResponse struct {
	Username string `json:"username"`
	Score    int    `json:"score"`
	// Rank and Players are left out if the rank lookup fails, so the score,
	// which has already been read or changed, still comes back.
	Rank    int `json:"rank,omitempty"`
	Players int `json:"players,omitempty"`
}

type leaderboardResponse struct {
	Leaderboard []db.LeaderboardEntry `json:"leaderboard"`
}

type roundResponse struct {
	Game string `json:"game"`
	games.Round
}

func (a *API) HelloWorldHandler(c *gin.Context) {
	c.JSON(http.StatusOK, returnBody{Hello: "Hello world!"})
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

	c.JSON(http.StatusOK, a.withStanding(ctx, tableName, scoreResponse{Username: username, Score: score}))
}

// withStanding adds the user's rank to resp. A failed lookup is recorded on
// the trace and leaves the rank out rather than failing the request.
func (a *API) withStanding(ctx context.Context, tableName string, resp scoreResponse) scoreResponse {
	st, err := a.store.GetStanding(ctx, tableName, resp.Username)
	if err != nil {
		o11y.AddFieldToTrace(ctx, "standing-error", err)
		return resp
	}
	resp.Rank, resp.Players = st.Rank, st.Players
	return resp
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

	if requestBody.Score == nil {
		c.JSON(http.StatusBadRequest, returnBody{Error: "score required"})
		return
	}

	sql, err := a.store.UpdateScoreForUser(requestBody.TableName, requestBody.User, *requestBody.Score, requestBody.Column, ctx)
	if errors.Is(err, db.ErrColumnNotUpdatable) || errors.Is(err, db.ErrNegativeScore) {
		c.JSON(http.StatusBadRequest, returnBody{Error: err.Error()})
		return
	}
	if errors.Is(err, db.ErrUserNotFound) {
		c.JSON(http.StatusNotFound, returnBody{Error: err.Error()})
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
	c.JSON(http.StatusOK, a.withStanding(ctx, requestBody.TableName, scoreResponse{Username: requestBody.User, Score: score}))
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
