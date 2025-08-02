package httpapi

import (
	"fmt"
	"github.com/circleci/ex/o11y"
	"github.com/gin-gonic/gin"
	"github.com/imlogang/api-service/cmd/db"
	"github.com/imlogang/api-service/cmd/games"
	"net/http"
)

type requestBody struct {
	TableName    string `json:"table_name"`
	User         string `json:"username"`
	Score        int    `json:"score"`
	Column       string `json:"column"`
	SecondColumn string `json:"second_column"`
	NumInArray   int    `json:"numinarray"`
	Answer       string `json:"answer"`
}

type returnBody struct {
	Hello        string   `json:"hello,omitempty"`
	Tables       []string `json:"tables,omitempty"`
	TableCreated string   `json:"table_created,omitempty"`
	TableDeleted string   `json:"table_deleted,omitempty"`
	UpdateAnswer string   `json:"update_answer,omitempty"`
	Error        string   `json:"error,omitempty"`
	AddedUser    string   `json:"added_user,omitempty"`
}

func (a *API) HelloWorldHandler(c *gin.Context) {
	c.JSON(http.StatusOK, returnBody{Hello: "Hello world!"})
}

func (a *API) ListTablesHandler(c *gin.Context) {
	ctx := c.Request.Context()

	tables, err := db.ListTables()
	if err != nil {
		c.JSON(http.StatusBadRequest, returnBody{Error: err.Error()})
		return
	}

	o11y.AddFieldToTrace(ctx, "list-tables", tables)
	o11y.AddFieldToTrace(ctx, "request-remoteaddr", c.Request.RemoteAddr)

	c.JSON(http.StatusOK, returnBody{Tables: tables})
}

func (a *API) CreateTableHandler(c *gin.Context) {
	var requestBody requestBody
	ctx := c.Request.Context()

	err := c.BindJSON(&requestBody)
	if err != nil {
		c.JSON(http.StatusBadRequest, returnBody{Error: err.Error()})
		return
	}

	sql, err := db.CreateTable(requestBody.TableName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, returnBody{Error: err.Error()})
		return
	}

	ctx, createTableSpan := o11y.StartSpan(ctx, "CreateTableHandler")
	defer o11y.End(createTableSpan, &err)
	o11y.AddFieldToTrace(ctx, "create-tables", sql)
	o11y.AddFieldToTrace(ctx, "request-remoteaddr", c.Request.RemoteAddr)

	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, returnBody{TableCreated: requestBody.TableName})
}

func (a *API) DeleteTableHandler(c *gin.Context) {
	var requestBody requestBody
	ctx := c.Request.Context()
	err := c.BindJSON(&requestBody)
	if err != nil {
		c.JSON(http.StatusBadRequest, returnBody{Error: err.Error()})
		return
	}

	sql, err := db.DeleteTable(requestBody.TableName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, returnBody{Error: err.Error()})
		return
	}

	ctx, deleteTableSpan := o11y.StartSpan(ctx, "DeleteTableHandler")
	defer o11y.End(deleteTableSpan, &err)
	o11y.AddFieldToTrace(ctx, "delete-tables", sql)
	o11y.AddFieldToTrace(ctx, "request-remoteaddr", c.Request.RemoteAddr)

	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, returnBody{TableDeleted: requestBody.TableName})
}

func (a *API) UpdateTableWithUserHandler(c *gin.Context) {
	var requestBody requestBody
	ctx := c.Request.Context()
	err := c.BindJSON(&requestBody)

	ctx, updateTableWithUserSpan := o11y.StartSpan(ctx, "UpdateTableWithUserHandler")
	defer o11y.End(updateTableWithUserSpan, &err)
	if err != nil {
		c.JSON(http.StatusBadRequest, returnBody{Error: err.Error()})
		return
	}

	_, err = db.UpdateTableWithUser(requestBody.TableName, requestBody.User)
	if err != nil {
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

	score, err := db.GetCurrentScore(tableName, username)
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

	sql, err := db.UpdateScoreForUser(requestBody.TableName, requestBody.User, requestBody.Score, requestBody.Column)
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

	pokemon, err := games.GetPokemon()
	if err != nil {
		o11y.AddFieldToTrace(ctx, "db-error", err)
		c.JSON(http.StatusInternalServerError, returnBody{Error: err.Error()})
		return
	}

	o11y.AddFieldToTrace(ctx, "pokemon", pokemon)
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, fmt.Sprintf("%s\n", pokemon))
}

func (a *API) ReadAnswerFromDBHandler(c *gin.Context) {
	ctx := c.Request.Context()
	tableName := c.Query("tablename")
	column := c.Query("colum")

	ctx, answerFromTableSpan := o11y.StartSpan(ctx, "ReadAnswerFromDBHandler")
	var err error
	defer o11y.End(answerFromTableSpan, &err)

	if tableName == "" || column == "" {
		o11y.AddFieldToTrace(ctx, "table-name", tableName)
		o11y.AddFieldToTrace(ctx, "colum-name", column)
		c.JSON(http.StatusBadRequest, returnBody{Error: "tablename or column required"})
		return
	}
	answer, err := db.ReadAnswerFromDB(tableName, column)
	if err != nil {
		o11y.AddFieldToTrace(ctx, "db-error", err)
		c.JSON(http.StatusInternalServerError, returnBody{Error: err.Error()})
		return
	}
	o11y.AddFieldToTrace(ctx, "answer", answer)
	o11y.AddFieldToTrace(ctx, "request-remoteaddr", c.Request.RemoteAddr)
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, answer)
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

	leaderboard, err := db.GetLeaderboard(tableName)
	if err != nil {
		o11y.AddFieldToTrace(ctx, "db-error", err)
		c.JSON(http.StatusInternalServerError, returnBody{Error: err.Error()})
		return
	}
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, fmt.Sprintf("\n%s", leaderboard))
}
