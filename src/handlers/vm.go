package handlers

import (
	"encoding/json"
	"fmt"
	"rm-thierry/Proxmox-API/src/manager"
)

type VMConfig struct {
	Node    string
	VMID    string
	Name    string
	Cores   string
	Memory  string
	Disk    string
	Net     string
	ISO     string
	OSType  string
	CPU     string
	Sockets string
}

type ISO struct {
	Debian = "local:iso/debian-12.8.0-amd64-netinst.iso",
	Ubuntu = "local:iso/ubuntu-20.04.4-live-server-amd64.iso",
	CentOS = "local:iso/CentOS-8.5.2111-x86_64-dvd1.iso",
}


func NewDefaultVMConfig() VMConfig {
	return VMConfig{
		Node:    "pve",
		Cores:   "1",
		Memory:  "2048",
		Disk:    "local",
		Net:     "vmbr0",
		ISO:     "local:iso/debian-12.8.0-amd64-netinst.iso",
		OSType:  "l26",
		CPU:     "host",
		Sockets: "1",
	}
}

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
		return nil, fmt.Errorf("error performing API call: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON response: %v", err)
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	return data, nil
}

func GetVMIDByName(apiManager *manager.APIManager, node string, vmname string) (string, error) {
	vms, err := GetVMS(apiManager, node)
	if err != nil {
		return "", err
	}

	for _, vm := range vms {
		if name, ok := vm["name"].(string); ok && name == vmname {
			if vmid, ok := vm["vmid"].(float64); ok {
				return fmt.Sprintf("%.0f", vmid), nil
			}
		}
	}

	return "", fmt.Errorf("VM not found")
}

func CreateVM(apiManager *manager.APIManager, config VMConfig) (map[string]interface{}, error) {
	if config.VMID == "" {
		return nil, fmt.Errorf("VMID is required")
	}
	if config.Name == "" {
		return nil, fmt.Errorf("VM name is required")
	}

	payload := map[string]interface{}{
		"vmid":     config.VMID,
		"name":     config.Name,
		"cores":    config.Cores,
		"memory":   config.Memory,
		"virtio0":  config.Disk + ":0",
		"net0":     "virtio,bridge=" + config.Net,
		"ostype":   config.OSType,
		"scsihw":   "virtio-scsi-pci",
		"bootdisk": "virtio0",
		"sockets":  config.Sockets,
		"cpu":      config.CPU,
	}

	if config.ISO != "" {
		payload["ide2"] = config.ISO + ",media=cdrom"
	}

	debugJSON, _ := json.MarshalIndent(payload, "", "  ")
	fmt.Printf("Sending payload to API:\n%s\n", string(debugJSON))

	response, err := apiManager.ApiCall("POST", fmt.Sprintf("/nodes/%s/qemu", config.Node), payload)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Raw API Response:\n%s\n", string(response))

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON response: %v", err)
	}

	if result == nil {
		return nil, fmt.Errorf("received empty response from API")
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
	response, err := apiManager.ApiCall("POST", fmt.Sprintf("/nodes/%s/qemu/%s/status/start", node, vmid), nil)
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
	response, err := apiManager.ApiCall("POST", fmt.Sprintf("/nodes/%s/qemu/%s/status/stop", node, vmid), nil)
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
