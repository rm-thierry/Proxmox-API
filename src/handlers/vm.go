package handlers

import (
	"encoding/json"
	"fmt"
	"rm-thierry/Proxmox-API/src/manager"
)

func GetVMS(apiManager *manager.APIManager, node string) ([]map[string]interface{}, error) {
	response, err := apiManager.ApiCall("GET", fmt.Sprintf("/nodes/%s/qemu", node), nil)
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

	vms := make([]map[string]interface{}, len(data))
	for i, item := range data {
		vms[i] = item.(map[string]interface{})
	}

	return vms, nil

}

func GetVM(apiManager *manager.APIManager, node string, vmid string) (map[string]interface{}, error) {
	response, err := apiManager.ApiCall("GET", fmt.Sprintf("/nodes/%s/qemu/%s/status/current", node, vmid), nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON response: %v", err)
	}

	return result, nil
}

func GetVMIDByName(apiManager *manager.APIManager, node string, vmname string) (string, error) {
	vms, err := GetVMS(apiManager, node)
	if err != nil {
		return "", err
	}

	for _, vm := range vms {
		if vm["name"] == vmname {
			return vm["vmid"].(string), nil
		}
	}

	return "", fmt.Errorf("VM not found")
}

func CreateVM(apiManager *manager.APIManager, node string, vmid string, vmname string, cores string, memory string, disk string, net string) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"vmid":   vmid,
		"vmname": vmname,
		"cores":  cores,
		"memory": memory,
		"disk":   disk,
		"net":    net,
	}

	response, err := apiManager.ApiCall("POST", fmt.Sprintf("/nodes/%s/qemu", node), payload)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON response: %v", err)
	}

	return result, nil
}

func DeleteVM(apiManager *manager.APIManager, node string, vmid string) (map[string]interface{}, error) {
	response, err := apiManager.ApiCall("DELETE", fmt.Sprintf("/nodes/%s/qemu/%s", node, vmid), nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON response: %v", err)
	}

	return result, nil
}

func StartVM(apiManager *manager.APIManager, node string, vmid string) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"state": "start",
	}

	response, err := apiManager.ApiCall("POST", fmt.Sprintf("/nodes/%s/qemu/%s/status/start", node, vmid), payload)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON response: %v", err)
	}

	return result, nil
}

func StopVM(apiManager *manager.APIManager, node string, vmid string) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"state": "stop",
	}

	response, err := apiManager.ApiCall("POST", fmt.Sprintf("/nodes/%s/qemu/%s/status/stop", node, vmid), payload)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON response: %v", err)
	}

	return result, nil
}
