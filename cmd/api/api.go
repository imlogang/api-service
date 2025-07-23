package httpapi

import (
	"encoding/json"
	"fmt"
	"github.com/circleci/ex/o11y"
	"github.com/gin-gonic/gin"
	"go-api/cmd/db"
	"go-api/cmd/deluge"
	"go-api/cmd/games"
	"io"
	"log"
	"net/http"
)

type TorrentRequest struct {
	Parameters struct {
		URL string `json:"url"`
	} `json:"parameters"`
}

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
}

func AddTorrentHandler(w http.ResponseWriter, r *http.Request) {
	// Read the request body and check for errors
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close() // Close the body after reading to avoid resource leak

	// Log the raw request body for debugging purposes
	log.Printf("Received request body: %s", string(bodyBytes))

	// Initialize the struct to hold the parsed request
	var req TorrentRequest

	// Decode the JSON request body into the struct using the read bytes
	err = json.Unmarshal(bodyBytes, &req)
	if err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Check if the URL is empty and return an error
	if req.Parameters.URL == "" {
		http.Error(w, "Torrent URL is required", http.StatusBadRequest)
		return
	}

	// Log the parsed torrent URL for debugging purposes
	log.Printf("Parsed torrent URL: %s", req.Parameters.URL)

	// Call the AddTorrentFile function from the deluge package with the parsed URL
	result, err := deluge.AuthAndDownloadTorrent(req.Parameters.URL)
	if err != nil {
		http.Error(w, "Error downloading torrent: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the result as JSON and check for any error during encoding
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a *API) HelloWorldHandler(c *gin.Context) {
	c.JSON(http.StatusOK, returnBody{Hello: "Hello world!"})
}

func (a *API) ListTablesHandler(c *gin.Context) {
	ctx := c.Request.Context()

	tables, err := db.ListTables()
	if err != nil {
		c.Status(http.StatusBadRequest)
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
		err = fmt.Errorf("invalid body: %s", err)
		c.JSON(http.StatusBadRequest, fmt.Sprintf("invalid body: %s", err))
		return
	}

	sql, err := db.CreateTable(requestBody.TableName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("error creating table: %s", err))
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
		c.JSON(http.StatusBadRequest, fmt.Sprintf("invalid body: %s", err))
		return
	}

	sql, err := db.DeleteTable(requestBody.TableName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("error deleting table: %s", err))
		return
	}

	ctx, deleteTableSpan := o11y.StartSpan(ctx, "DeleteTableHandler")
	defer o11y.End(deleteTableSpan, &err)
	o11y.AddFieldToTrace(ctx, "delete-tables", sql)
	o11y.AddFieldToTrace(ctx, "request-remoteaddr", c.Request.RemoteAddr)

	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, returnBody{TableDeleted: requestBody.TableName})
}

func UpdateTableWithUser(w http.ResponseWriter, r *http.Request) {
	var requestBody requestBody
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	sql, err := db.UpdateTableWithUser(requestBody.TableName, requestBody.User)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"table_updated_with": sql}); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
	}
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
		c.JSON(http.StatusBadRequest, fmt.Sprintf("tablename: %s, or username: %s, cannot be empty", tableName, username))
		return
	}

	score, err := db.GetCurrentScore(tableName, username)
	if err != nil {
		o11y.AddFieldToTrace(ctx, "db-error", err)
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("error getting current score: %s", err))
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
		c.JSON(http.StatusBadRequest, fmt.Sprintf("Invalid request body: %s", err))
		return
	}

	sql, err := db.UpdateScoreForUser(requestBody.TableName, requestBody.User, requestBody.Score, requestBody.Column)
	if err != nil {
		o11y.AddFieldToTrace(ctx, "db-error", err)
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("error updating score: %s", err))
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
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("error getting pokemon: %s", err))
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
		c.JSON(http.StatusBadRequest, fmt.Sprintf("tablename: %s, or column: %s, cannot be empty", tableName, column))
		return
	}
	answer, err := db.ReadAnswerFromDB(tableName, column)
	if err != nil {
		o11y.AddFieldToTrace(ctx, "db-error", err)
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("error finding answer: %s", err))
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
		c.JSON(http.StatusBadRequest, fmt.Sprintf("tablename: %s, cannot be empty", tableName))
		return
	}

	leaderboard, err := db.GetLeaderboard(tableName)
	if err != nil {
		o11y.AddFieldToTrace(ctx, "db-error", err)
		c.JSON(http.StatusNotFound, fmt.Sprintf("error getting leaderboard: %s", err))
		return
	}
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, leaderboard)
}
