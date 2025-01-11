package handlers

import (
	"encoding/json"
	"fmt"
	"rm-thierry/Proxmox-API/src/manager"
)

func GetNodes(apiManager *manager.APIManager) ([]map[string]interface{}, error) {
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
