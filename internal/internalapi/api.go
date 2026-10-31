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
	Game string `json:"game"`
	// TableName is the old way to pick the game; see scoreTable.
	TableName string `json:"table_name"`
	User      string `json:"username"`
	// Score is a pointer so that leaving it out is an error, not a reset to 0.
	Score  *int   `json:"score"`
	Column string `json:"column"`
	// Points is how much increment_score adds. It defaults to 1.
	Points *int `json:"points"`
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

type collectRequest struct {
	Game     string `json:"game"`
	Username string `json:"username"`
	ID       int    `json:"id"`
	Shiny    bool   `json:"shiny"`
}

type collectionResponse struct {
	Username string `json:"username"`
	Game     string `json:"game"`
	// New is set by add_to_collection when the item wasn't in the
	// collection before.
	New         bool `json:"new,omitempty"`
	Caught      int  `json:"caught"`
	ShinyCaught int  `json:"shiny_caught"`
	// Total is how many items the game has, so Caught == Total is complete.
	Total int `json:"total"`
}

type roundResponse struct {
	Game string `json:"game"`
	games.Round
}

func (a *API) HelloWorldHandler(c *gin.Context) {
	c.JSON(http.StatusOK, returnBody{Hello: "Hello world!"})
}

// scoreTable returns the scores table for a game. Callers name the game;
// table names from callers are only still accepted, for bots that predate
// game, if they are a known game's table. Either way only tables that belong
// to a game the API serves can be reached.
func (a *API) scoreTable(game, tableName string) (string, error) {
	if game != "" {
		if _, ok := a.games[game]; !ok {
			return "", fmt.Errorf("unknown game %q", game)
		}
		return game + "_scores", nil
	}
	if tableName != "" {
		for name := range a.games {
			if tableName == name+"_scores" {
				return tableName, nil
			}
		}
		return "", fmt.Errorf("unknown table %q", tableName)
	}
	return "", errors.New("game required")
}

func (a *API) GetScoreHandler(c *gin.Context) {
	ctx := c.Request.Context()

	username := c.Query("username")

	var err error
	ctx, getScoreHandlerSpan := o11y.StartSpan(ctx, "GetScoreHandler")
	defer o11y.End(getScoreHandlerSpan, &err)
	tableName, err := a.scoreTable(c.Query("game"), c.Query("tablename"))
	o11y.AddFieldToTrace(ctx, "table_name", tableName)
	if err != nil {
		c.JSON(http.StatusBadRequest, returnBody{Error: err.Error()})
		return
	}
	if username == "" {
		c.JSON(http.StatusBadRequest, returnBody{Error: "username required"})
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
	tableName, err := a.scoreTable(requestBody.Game, requestBody.TableName)
	if err != nil {
		c.JSON(http.StatusBadRequest, returnBody{Error: err.Error()})
		return
	}

	sql, err := a.store.UpdateScoreForUser(tableName, requestBody.User, *requestBody.Score, requestBody.Column, ctx)
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

	var err error
	ctx, leaderboardHandlerSpan := o11y.StartSpan(ctx, "LeaderboardHandler")
	defer o11y.End(leaderboardHandlerSpan, &err)

	tableName, err := a.scoreTable(c.Query("game"), c.Query("tablename"))
	o11y.AddFieldToTrace(ctx, "table_name", tableName)
	if err != nil {
		c.JSON(http.StatusBadRequest, returnBody{Error: err.Error()})
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

// IncrementScoreHandler adds points (1 unless given) to a user's score
// atomically and returns the new score.
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
	tableName, err := a.scoreTable(requestBody.Game, requestBody.TableName)
	if err != nil {
		c.JSON(http.StatusBadRequest, returnBody{Error: err.Error()})
		return
	}
	if requestBody.User == "" {
		c.JSON(http.StatusBadRequest, returnBody{Error: "username required"})
		return
	}

	points := 1
	if requestBody.Points != nil {
		points = *requestBody.Points
	}
	score, err := a.store.IncrementScore(ctx, tableName, requestBody.User, points)
	if errors.Is(err, db.ErrBadPoints) {
		c.JSON(http.StatusBadRequest, returnBody{Error: err.Error()})
		return
	}
	if err != nil {
		o11y.AddFieldToTrace(ctx, "db-error", err)
		c.JSON(http.StatusInternalServerError, returnBody{Error: err.Error()})
		return
	}
	o11y.AddFieldToTrace(ctx, "score", score)
	c.JSON(http.StatusOK, a.withStanding(ctx, tableName, scoreResponse{Username: requestBody.User, Score: score}))
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

// AddToCollectionHandler records that a user guessed a round's answer, and
// returns how much of the game's collection they now have.
func (a *API) AddToCollectionHandler(c *gin.Context) {
	ctx := c.Request.Context()
	var req collectRequest
	err := c.BindJSON(&req)
	ctx, span := o11y.StartSpan(ctx, "AddToCollectionHandler")
	defer o11y.End(span, &err)

	if err != nil {
		c.JSON(http.StatusBadRequest, returnBody{Error: err.Error()})
		return
	}
	if req.Game == "" || req.Username == "" || req.ID == 0 {
		c.JSON(http.StatusBadRequest, returnBody{Error: "game, username and id required"})
		return
	}
	game, ok := a.games[req.Game]
	if !ok {
		c.JSON(http.StatusBadRequest, returnBody{Error: fmt.Sprintf("unknown game %q", req.Game)})
		return
	}
	o11y.AddFieldToTrace(ctx, "game", req.Game)
	o11y.AddFieldToTrace(ctx, "item_id", req.ID)

	added, err := a.store.AddToCollection(ctx, req.Game, req.Username, req.ID, req.Shiny)
	if err != nil {
		o11y.AddFieldToTrace(ctx, "db-error", err)
		c.JSON(http.StatusInternalServerError, returnBody{Error: err.Error()})
		return
	}
	resp, err := a.collection(ctx, game, req.Game, req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, returnBody{Error: err.Error()})
		return
	}
	resp.New = added
	c.JSON(http.StatusOK, resp)
}

// GetCollectionHandler returns how much of a game's collection a user has.
func (a *API) GetCollectionHandler(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Query("game")
	username := c.Query("username")

	var err error
	ctx, span := o11y.StartSpan(ctx, "GetCollectionHandler")
	defer o11y.End(span, &err)

	if name == "" || username == "" {
		c.JSON(http.StatusBadRequest, returnBody{Error: "game and username required"})
		return
	}
	game, ok := a.games[name]
	if !ok {
		c.JSON(http.StatusBadRequest, returnBody{Error: fmt.Sprintf("unknown game %q", name)})
		return
	}

	resp, err := a.collection(ctx, game, name, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, returnBody{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (a *API) collection(ctx context.Context, game games.Game, name, username string) (collectionResponse, error) {
	col, err := a.store.GetCollection(ctx, name, username)
	if err != nil {
		return collectionResponse{}, err
	}
	total, err := game.Size(ctx)
	if err != nil {
		return collectionResponse{}, err
	}
	return collectionResponse{
		Username:    username,
		Game:        name,
		Caught:      col.Caught,
		ShinyCaught: col.Shiny,
		Total:       total,
	}, nil
}
