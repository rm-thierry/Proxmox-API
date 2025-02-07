package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	err := godotenv.Load("env/.env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	baseURL := os.Getenv("APIURL")
	if baseURL == "" {
		log.Fatalf("APIURL not set in .env file")
	}

	return &APIManager{
		BaseURL:     baseURL,
		Node:        os.Getenv("NODE"),
		TokenID:     os.Getenv("PROXMOX_TOKEN_ID"),
		TokenSecret: os.Getenv("PROXMOX_TOKEN_SECRET"),
	}
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

	req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s=%s", manager.TokenID, manager.TokenSecret))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing request: %v", err)
	}
	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", responseBody)
	}

	return responseBody, nil
}
