package handlers

import (
	"encoding/json"
	"fmt"
	"rm-thierry/Proxmox-API/src/manager"
)

func GetNodess(apiManager *manager.APIManager) ([]map[string]interface{}, error) {
	response, err := apiManager.ApiCall("GET", "/nodes", nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON response: %v", err)
	}

	data, ok := result["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	nodes := make([]map[string]interface{}, len(data))
	for i, item := range data {
		nodes[i] = item.(map[string]interface{})
	}

	return nodes, nil
}

func GetNode(apiManager *manager.APIManager, node string) (map[string]interface{}, error) {
	response, err := apiManager.ApiCall("GET", fmt.Sprintf("/nodes/%s", node), nil)
	if err != nil {
		return nil, fmt.Errorf("error performing API call: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("error parsing JSON response: %w", err)
	}

	data, ok := result["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format: 'data' field is missing or incorrect")
	}

	if len(data) > 0 {
		if nodeData, ok := data[0].(map[string]interface{}); ok {
			return nodeData, nil
		}
	}

	return nil, fmt.Errorf("no valid node data found")
}
