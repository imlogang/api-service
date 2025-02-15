package deluge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"io"
)

// Struct for the JSON-RPC request
type JsonRpcRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// Struct for the JSON-RPC response
type JsonRpcResponse struct {
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
	ID     int         `json:"id"`
}

func AddTorrentFile(torrentPath string, downloadDir string) (interface{}, error) {
	// Define the JSON-RPC request
	req := JsonRpcRequest{
		Jsonrpc: "2.0",
		Method:  "core.add_torrent_file",
		Params:  []interface{}{torrentPath, downloadDir},
		ID:      1,
	}

	// Marshal the request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	// Send the request to the Deluge server
	resp, err := http.Post("http://deluge.logangodsey.com:8112/json", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error sending request to Deluge: %w", err)
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
		return nil, fmt.Errorf("Deluge API error: %v", jsonResponse.Error)
	}

	return jsonResponse.Result, nil
}
