package httpapi

import (
	"encoding/json"
	"fmt"
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

func HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	// Write the response and check for any error, but discard 'n' as it's not needed
	_, err := w.Write([]byte("Hello, World!"))
	if err != nil {
		// If there is an error, return an internal server error response
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func ListTablesAPI(w http.ResponseWriter, r *http.Request) {
	tables, err := db.ListTables()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error listing tables: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the tables as a JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tables); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode tables: %v", err), http.StatusInternalServerError)
	}
}

func CreateTableAPI(w http.ResponseWriter, r *http.Request) {
	var requestBody requestBody
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	sql, err := db.CreateTable(requestBody.TableName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"table_name_created": sql}); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
	}
}

func DeleteTableAPI(w http.ResponseWriter, r *http.Request) {
	var requestBody requestBody
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	sql, err := db.DeleteTable(requestBody.TableName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"table_name_deleted": sql}); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
	}
}

func UpdateTableWithUser(w http.ResponseWriter, r *http.Request) {
	var requestBody struct {
		TableName string `json:"table_name"`
		User      string `json:"username"`
	}
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

func GetScoreAPI(w http.ResponseWriter, r *http.Request) {
	fullURL := r.URL.String()
	tableName := r.URL.Query().Get("tablename")
	username := r.URL.Query().Get("username")

	if tableName == "" || username == "" {
		http.Error(w, fmt.Sprintf("Error: tableName or username cannot be empty. Received - tableName: %s, username: %s\n The full URL: %s", tableName, username, fullURL), http.StatusBadRequest)
		return
	}

	score, err := db.GetCurrentScore(tableName, username)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving score: %s", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "Score for %s: %d\n", username, score)
}

func UpdateScoreForUserAPI(w http.ResponseWriter, r *http.Request) {
	var requestBody struct {
		TableName string `json:"table_name"`
		User      string `json:"username"`
		Score     int    `json:"score"`
		Column    string `json:"column"`
	}
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	sql, err := db.UpdateScoreForUser(requestBody.TableName, requestBody.User, requestBody.Score, requestBody.Column)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"update_answer:": sql}); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
	}
}

func GetPokemonAPI(w http.ResponseWriter, r *http.Request) {
	pokemon, err := games.GetPokemon()
	if err != nil {
		http.Error(w, fmt.Sprintf("there was an error finding your pokemon, %s", err), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "%s\n", pokemon)
}

func PutAnswerInDBAPI(w http.ResponseWriter, r *http.Request) {
	var requestBody struct {
		TableName    string `json:"table_name"`
		Answer       string `json:"answer"`
		Column       string `json:"column"`
		SecondColumn string `json:"second_column"`
		NumInArray   int    `json:"numinarray"`
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		log.Printf("invalid request body: %s", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	sql, err := db.PutAnswerInDB(requestBody.TableName, requestBody.Answer, requestBody.Column, requestBody.SecondColumn, requestBody.NumInArray)
	if err != nil {
		log.Printf("could not put in database: %s", err)
		http.Error(w, fmt.Sprintf(`{"error": "Invalid request body: %v"}`, err), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"answer": sql}); err != nil {
		log.Printf("error updating answer: %s", err)
		http.Error(w, fmt.Sprintf(`{"error": "Error updating answer: %v"}`, err), http.StatusBadRequest)
	}
}

func ReadAnswerFromDBAPI(w http.ResponseWriter, r *http.Request) {
	tableName := r.URL.Query().Get("tablename")
	column := r.URL.Query().Get("column")
	if tableName == "" || column == "" {
		http.Error(w, fmt.Sprintf("tablename: %s or column: %s cannot be empty.", tableName, column), http.StatusBadRequest)
		return
	}
	answer, err := db.ReadAnswerFromDB(tableName, column)
	if err != nil {
		http.Error(w, fmt.Sprintf("there was an error finding the answer: %s", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "%s\n", answer)
}

func LeaderboardAPI(w http.ResponseWriter, r *http.Request) {
	fullURL := r.URL.String()
	tableName := r.URL.Query().Get("tablename")

	if tableName == "" {
		http.Error(w, fmt.Sprintf("Error: tableName cannot be empty. Received - tableName: %s\n The full URL: %s", tableName, fullURL), http.StatusBadRequest)
		return
	}

	leaderboard, err := db.GetLeaderboard(tableName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving leaderboard: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "Leaderboard:\n%s", leaderboard)
}
