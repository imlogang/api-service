package deluge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

var delugeURL = "https://deluge.logangodsey.com/json" // Update this with your Deluge API URL
var hostID string

func AddHostAndConnect() error {
	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")

	if username == "" || password == "" {
		return fmt.Errorf("USERNAME or PASSWORD environment variables not set")
	}

	// Step 1: Authenticate with Deluge
	authReq := JsonRpcRequest{
		Jsonrpc:  "2.0",
		Method:   "auth.login",
		Params:   []interface{}{password},
		ID:       1,
	}

	reqBody, err := json.Marshal(authReq)
	if err != nil {
		return fmt.Errorf("error marshaling auth request: %w", err)
	}

	// Send the authentication request
	resp, err := http.Post("https://deluge.logangodsey.com/json", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("error sending auth request to Deluge: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	var authResponse JsonRpcResponse
	err = json.Unmarshal(body, &authResponse)
	if err != nil {
		return fmt.Errorf("error unmarshaling auth response: %w", err)
	}

	// Check for authentication error
	if authResponse.Error != nil {
		return fmt.Errorf("authentication failed: %v", authResponse.Error)
	}

	// Step 2: Add the host after successful authentication
	addHostReq := JsonRpcRequest{
		Jsonrpc:  "2.0",
		Method:   "web.add_host",
		Params:   []interface{}{password},
		ID:       1,
		Username: username,
		Password: password,
	}

	reqBody, err = json.Marshal(addHostReq)
	if err != nil {
		return fmt.Errorf("error marshaling add host request: %w", err)
	}

	// Send the add host request
	resp, err = http.Post("https://deluge.logangodsey.com/json", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("error sending add host request to Deluge: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	var addHostResponse JsonRpcResponse
	err = json.Unmarshal(body, &addHostResponse)
	if err != nil {
		return fmt.Errorf("error unmarshaling add host response: %w", err)
	}

	// Handle any error from the add_host call
	if addHostResponse.Error != nil {
		return fmt.Errorf("add host error: %v", addHostResponse.Error)
	}

	// Save the host ID for later use
	hostID = addHostResponse.Result.(string)

	// Successful connection
	return nil
}

func AddTorrentFile(torrentPath string) (interface{}, error) {
	// Add authentication and connection setup first
	if err := AddHostAndConnect(); err != nil {
		return nil, fmt.Errorf("failed to authenticate and connect: %w", err)
	}

	// Now proceed with adding the torrent file
	req := JsonRpcRequest{
		Jsonrpc: "2.0",
		Method:  "web.download_torrent_from_url",
		Params:  []interface{}{torrentPath},
		ID:      3,
	}

	// Marshal the request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling add torrent request: %w", err)
	}

	// Send the request to the Deluge server
	resp, err := http.Post(delugeURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error sending add torrent request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	// Parse the JSON-RPC response
	var jsonResponse JsonRpcResponse
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	// Return the result
	if jsonResponse.Error != nil {
		return nil, fmt.Errorf("deluge API error: %v", jsonResponse.Error)
	}

	return jsonResponse.Result, nil
}
