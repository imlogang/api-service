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

func AddHostAndConnect() error {
	// Retrieve username and password from environment variables
	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")

	if username == "" || password == "" {
		return fmt.Errorf("USERNAME or PASSWORD environment variables not set")
	}

	// Define the JSON-RPC request to add a host
	addHostReq := JsonRpcRequest{
		Jsonrpc:  "2.0",
		Method:   "web.add_host",
		Params:   []interface{}{"https://deluge.logangodsey.com/json", username, password}, // Replace 127.0.0.1 with your Deluge host IP
		ID:       1,
		Username: username,
		Password: password,
	}

	reqBody, err := json.Marshal(addHostReq)
	if err != nil {
		return fmt.Errorf("error marshaling add host request: %w", err)
	}

	// Send the request to the Deluge server
	resp, err := http.Post(delugeURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("error sending add host request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading add host response: %w", err)
	}

	// Parse the response
	var jsonResponse JsonRpcResponse
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return fmt.Errorf("error unmarshaling add host response: %w", err)
	}

	// Check for errors
	if jsonResponse.Error != nil {
		return fmt.Errorf("add host error: %v", jsonResponse.Error)
	}

	// Now that the host is added, connect to it
	hostID := jsonResponse.Result // Assuming host ID is returned as the result

	connectReq := JsonRpcRequest{
		Jsonrpc: "2.0",
		Method:  "web.connect",
		Params:  []interface{}{hostID},
		ID:      2,
	}

	reqBody, err = json.Marshal(connectReq)
	if err != nil {
		return fmt.Errorf("error marshaling connect request: %w", err)
	}

	// Send the connection request to the Deluge server
	resp, err = http.Post(delugeURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("error sending connect request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading connect response: %w", err)
	}

	// Parse the response
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return fmt.Errorf("error unmarshaling connect response: %w", err)
	}

	// Check if the connection is successful
	if jsonResponse.Error != nil {
		return fmt.Errorf("connect error: %v", jsonResponse.Error)
	}

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
