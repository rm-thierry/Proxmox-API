package manager

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
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
	_ = godotenv.Load("env/.env")

	baseURL := os.Getenv("APIURL")
	if baseURL == "" {
		baseURL = "https://localhost:8006/api2/json"
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

	if node != "" && tokenID != "" && tokenSecret != "" {
		response, err := apiManager.ApiCall("GET", "/nodes", nil)
		if err == nil {
			var result map[string]interface{}
			if err := json.Unmarshal(response, &result); err == nil {
				if data, ok := result["data"].([]interface{}); ok && len(data) > 0 {
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
						apiManager.Node = availableNodes[0]
					}
				}
			}
		}
	}

	return apiManager
}

func (manager *APIManager) ApiCall(method, endpoint string, payload interface{}) ([]byte, error) {
	return manager.ApiCallWithOptions(method, endpoint, payload, true)
}

func (manager *APIManager) ApiCallWithOptions(method, endpoint string, payload interface{}, useJsonContentType bool) ([]byte, error) {
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

	if manager.TokenID != "" && manager.TokenSecret != "" {
		req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s=%s", manager.TokenID, manager.TokenSecret))
	}

	if useJsonContentType && (method == "POST" || method == "PUT" || payload != nil) {
		req.Header.Set("Content-Type", "application/json")
	}

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
		var errorDetails string

		var errorResponse map[string]interface{}
		if err := json.Unmarshal(responseBody, &errorResponse); err == nil {
			if errData, ok := errorResponse["errors"]; ok {
				jsonErrData, _ := json.Marshal(errData)
				errorDetails = string(jsonErrData)
			} else if data, ok := errorResponse["data"]; ok {
				if dataObj, ok := data.(map[string]interface{}); ok {
					if msg, ok := dataObj["msg"].(string); ok {
						errorDetails = msg
					}
				}

				if data == nil && resp.StatusCode == 500 {
					errorDetails = "Proxmox API returned an internal server error. This could be due to invalid VM parameters, " +
						"insufficient disk space, missing privileges, or an issue with the storage configuration."
				}
			}
		}

		if errorDetails != "" {
			return nil, fmt.Errorf("API error (Status %d): %s - Details: %s", resp.StatusCode, responseBody, errorDetails)
		} else {
			return nil, fmt.Errorf("API error (Status %d): %s", resp.StatusCode, responseBody)
		}
	}

	return responseBody, nil
}
