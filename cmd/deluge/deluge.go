package deluge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
)

// Struct for the JSON-RPC request
type JsonRpcRequest struct {
	Jsonrpc  string        `json:"jsonrpc"`
	Method   string        `json:"method"`
	Params   []interface{} `json:"params"`
	ID       int           `json:"id"`
	Username string        `json:"username,omitempty"`
	Password string        `json:"password,omitempty"`
}

// Struct for the JSON-RPC response
type JsonRpcResponse struct {
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
	ID     int         `json:"id"`
}

func createClientWithCookies() (*http.Client, error) {
	// Create a CookieJar to store cookies
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("error creating cookie jar: %w", err)
	}

	// Create an HTTP client with the CookieJar
	client := &http.Client{
		Jar: jar,
	}

	return client, nil
}

func AuthAndDownloadTorrent(torrentPath string) (interface{}, error) {
	// Get credentials from environment variables
	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")
	if username == "" || password == "" {
		log.Fatal("USERNAME or PASSWORD environment variables not set")
	}

	// Create a client with cookie jar to maintain the session
	client, err := createClientWithCookies()
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	// Define the authentication request
	authReq := JsonRpcRequest{
		Jsonrpc: "2.0",
		Method:  "auth.login",
		Params:  []interface{}{password}, // Only password as the param
		ID:      1,
	}

	// Marshal the request to JSON
	reqBody, err := json.Marshal(authReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling auth request: %w", err)
	}

	// Send the authentication request
	resp, err := client.Post("https://deluge.logangodsey.com/json", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error sending auth request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading auth response: %w", err)
	}

	// Parse the authentication response
	var jsonResponse JsonRpcResponse
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling auth response: %w", err)
	}

	// Check for authentication errors
	if jsonResponse.Error != nil {
		return nil, fmt.Errorf("authentication error: %v", jsonResponse.Error)
	}

	// Define the request for downloading the torrent
	torrentReq := JsonRpcRequest{
		Jsonrpc: "2.0",
		Method:  "web.download_torrent_from_url",
		Params:  []interface{}{torrentPath},
		ID:      2,
	}

	// Marshal the torrent request to JSON
	reqBody, err = json.Marshal(torrentReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling torrent request: %w", err)
	}

	// Send the request to download the torrent
	resp, err = client.Post("https://deluge.logangodsey.com/json", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error sending download request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading download response: %w", err)
	}

	// Parse the download response
	err = json.Unmarshal(body, &jsonResponse)
	fmt.Println(&jsonResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling download response: %w", err)
	}

	// Check if there is an error in the response
	if jsonResponse.Error != nil {
		return nil, fmt.Errorf("error downloading torrent: %v", jsonResponse.Error)
	}

	// Step 1: Get the path of the downloaded torrent
	torrentFilePath := jsonResponse.Result.(string)

	// Step 2: Add the downloaded torrent to Deluge using web.add_torrents
	addTorrentReq := JsonRpcRequest{
		Jsonrpc: "2.0",
		Method:  "web.add_torrents",
		Params: []interface{}{
			[]interface{}{
				map[string]interface{}{
					"path": torrentFilePath,
					"options": map[string]interface{}{
						"file_priorities": []int{1, 1, 1, 1, 1},
						"add_paused":      false,
					},
				},
			},
		},
		ID:       3,
		Username: username,
		Password: password,
	}

	// Marshal the add torrent request to JSON
	reqBody, err = json.Marshal(addTorrentReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling add torrent request: %w", err)
	}

	// Send the request to add the torrent to Deluge
	resp, err = client.Post("https://deluge.logangodsey.com/json", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error sending add torrent request to Deluge: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading add torrent response: %w", err)
	}

	// Parse the add torrent response
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling add torrent response: %w", err)
	}

	// Check if there is an error in the response
	if jsonResponse.Error != nil {
		return nil, fmt.Errorf("deluge API error: %v", jsonResponse.Error)
	}

	// Return the result (torrent added successfully)
	fmt.Println(jsonResponse.Result)
	fmt.Println(jsonResponse.ID)
	return jsonResponse.Result, nil
}
