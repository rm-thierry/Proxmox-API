package manager

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type APIManager struct {
	BaseURL     string
	Node        string
	TokenID     string
	TokenSecret string
}

func NewAPIManager() *APIManager {
	// Try to load environment variables but don't fail if not found
	_ = godotenv.Load("env/.env")

	baseURL := os.Getenv("APIURL")
	if baseURL == "" {
		log.Println("Warning: APIURL not set in environment")
		baseURL = "https://localhost:8006/api2/json" // Default value
	}

	node := os.Getenv("NODE")
	tokenID := os.Getenv("PROXMOX_TOKEN_ID")
	tokenSecret := os.Getenv("PROXMOX_TOKEN_SECRET")

	apiManager := &APIManager{
		BaseURL:     baseURL,
		Node:        node,
		TokenID:     tokenID,
		TokenSecret: tokenSecret,
	}

	// Try to validate and possibly correct the node if it seems wrong
	if node != "" && tokenID != "" && tokenSecret != "" {
		// Attempt to get a list of nodes from the cluster
		response, err := apiManager.ApiCall("GET", "/nodes", nil)
		if err != nil {
			log.Printf("Warning: Could not verify nodes: %v", err)
		} else {
			var result map[string]interface{}
			if err := json.Unmarshal(response, &result); err == nil {
				if data, ok := result["data"].([]interface{}); ok && len(data) > 0 {
					// Check if current node is valid
					nodeValid := false
					var availableNodes []string

					for _, n := range data {
						if nodeInfo, ok := n.(map[string]interface{}); ok {
							if nodeName, ok := nodeInfo["node"].(string); ok {
								availableNodes = append(availableNodes, nodeName)
								if nodeName == node {
									nodeValid = true
								}
							}
						}
					}

					if !nodeValid && len(availableNodes) > 0 {
						// If current node is invalid but we have other nodes, use the first one
						log.Printf("Warning: Node '%s' not found. Using '%s' instead.", node, availableNodes[0])
						apiManager.Node = availableNodes[0]
					}
				}
			}
		}
	}

	return apiManager
}

func (manager *APIManager) ApiCall(method, endpoint string, payload interface{}) ([]byte, error) {
	url := manager.BaseURL + endpoint

	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("error encoding payload: %v", err)
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set authentication headers if token credentials are available
	if manager.TokenID != "" && manager.TokenSecret != "" {
		req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s=%s", manager.TokenID, manager.TokenSecret))
	}
	
	req.Header.Set("Content-Type", "application/json")

	// Create a client that skips TLS verification to handle self-signed certificates
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing request to %s: %v", url, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (Status %d): %s", resp.StatusCode, responseBody)
	}

	return responseBody, nil
}
