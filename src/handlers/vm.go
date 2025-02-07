package handlers

import (
	"encoding/json"
	"fmt"
	"rm-thierry/Proxmox-API/src/manager"
	"strings"
)

type VMConfig struct {
	Node    string `json:"node"`
	VMID    string `json:"vmid"`
	Name    string `json:"name"`
	Cores   string `json:"cores"`
	Memory  string `json:"memory"`
	Disk    string `json:"disk"`
	Net     string `json:"net"`
	ISO     string `json:"iso"`
	OSType  string `json:"ostype"`
	CPU     string `json:"cpu"`
	Sockets string `json:"sockets"`
}

type ISO struct {
	Debian  string `json:"debian"`
	Ubuntu  string `json:"ubuntu"`
	Windows string `json:"windows"`
}

func GetISOs() ISO {
	return ISO{
		Debian:  "local:iso/debian-12.9.0-amd64-netinst.iso",
		Ubuntu:  "local:iso/ubuntu-22.04.3-live-server-amd64.iso",
		Windows: "local:iso/windows-server-2022.iso",
	}
}

func NewDefaultVMConfig() VMConfig {
	return VMConfig{
		Node:    manager.NewAPIManager().Node,
		Cores:   "1",
		Memory:  "2048",
		Disk:    "local",
		Net:     "vmbr0",
		ISO:     GetISOs().Debian,
		OSType:  "l26",
		CPU:     "host",
		Sockets: "1",
	}
}

func validateVM(apiManager *manager.APIManager, config VMConfig) error {
	if config.VMID == "" || config.Name == "" {
		return fmt.Errorf("VMID and Name are required")
	}

	exists, err := checkVMExists(apiManager, config.Node, config.VMID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("VM with ID %s already exists", config.VMID)
	}

	if err := validateStorage(apiManager, config); err != nil {
		return err
	}

	if err := validateNetwork(apiManager, config); err != nil {
		return err
	}

	return nil
}

func checkVMExists(apiManager *manager.APIManager, node, vmid string) (bool, error) {
	vms, err := GetVMS(apiManager, node)
	if err != nil {
		return false, err
	}

	for _, vm := range vms {
		if id, ok := vm["vmid"].(float64); ok {
			if fmt.Sprintf("%.0f", id) == vmid {
				return true, nil
			}
		}
	}
	return false, nil
}

func validateStorage(apiManager *manager.APIManager, config VMConfig) error {
	storage := strings.Split(config.Disk, ":")[0]
	response, err := apiManager.ApiCall("GET", fmt.Sprintf("/nodes/%s/storage", config.Node), nil)
	if err != nil {
		return fmt.Errorf("failed to check storage: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return err
	}

	data, ok := result["data"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid storage response")
	}

	for _, s := range data {
		if store, ok := s.(map[string]interface{}); ok {
			if store["storage"].(string) == storage {
				return nil
			}
		}
	}
	return fmt.Errorf("storage %s not found", storage)
}

func validateNetwork(apiManager *manager.APIManager, config VMConfig) error {
	response, err := apiManager.ApiCall("GET", fmt.Sprintf("/nodes/%s/network", config.Node), nil)
	if err != nil {
		return fmt.Errorf("failed to check network: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return err
	}

	data, ok := result["data"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid network response")
	}

	for _, n := range data {
		if net, ok := n.(map[string]interface{}); ok {
			if net["iface"].(string) == config.Net {
				return nil
			}
		}
	}
	return fmt.Errorf("network bridge %s not found", config.Net)
}

func buildVMPayload(config VMConfig) map[string]interface{} {
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

	return payload
}

func parseAPIResponse(response []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %v", err)
	}

	if result == nil {
		return nil, fmt.Errorf("empty API response")
	}

	return result, nil
}

func GetVMS(apiManager *manager.APIManager, node string) ([]map[string]interface{}, error) {
	response, err := apiManager.ApiCall("GET", fmt.Sprintf("/nodes/%s/qemu", node), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get VMs: %v", err)
	}

	result, err := parseAPIResponse(response)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("failed to get VM: %v", err)
	}

	result, err := parseAPIResponse(response)
	if err != nil {
		return nil, err
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
	if err := validateVM(apiManager, config); err != nil {
		return nil, err
	}

	payload := buildVMPayload(config)
	response, err := apiManager.ApiCall("POST", fmt.Sprintf("/nodes/%s/qemu", config.Node), payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM: %v", err)
	}

	return parseAPIResponse(response)
}

func DeleteVM(apiManager *manager.APIManager, node string, vmid string) (map[string]interface{}, error) {
	exists, err := checkVMExists(apiManager, node, vmid)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("VM with ID %s does not exist", vmid)
	}

	response, err := apiManager.ApiCall("DELETE", fmt.Sprintf("/nodes/%s/qemu/%s", node, vmid), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to delete VM: %v", err)
	}

	return parseAPIResponse(response)
}

func StartVM(apiManager *manager.APIManager, node string, vmid string) (map[string]interface{}, error) {
	exists, err := checkVMExists(apiManager, node, vmid)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("VM with ID %s does not exist", vmid)
	}

	response, err := apiManager.ApiCall("POST", fmt.Sprintf("/nodes/%s/qemu/%s/status/start", node, vmid), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start VM: %v", err)
	}

	return parseAPIResponse(response)
}

func StopVM(apiManager *manager.APIManager, node string, vmid string) (map[string]interface{}, error) {
	exists, err := checkVMExists(apiManager, node, vmid)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("VM with ID %s does not exist", vmid)
	}

	response, err := apiManager.ApiCall("POST", fmt.Sprintf("/nodes/%s/qemu/%s/status/stop", node, vmid), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to stop VM: %v", err)
	}

	return parseAPIResponse(response)
}

func GetHighestVMID(apiManager *manager.APIManager, node string) (int, error) {
	response, err := apiManager.ApiCall("GET", fmt.Sprintf("/nodes/%s/qemu", node), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get VMs: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	data, ok := result["data"].([]interface{})
	if !ok {
		return 101, nil
	}

	highest := 100
	for _, item := range data {
		vm, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		if vmid, ok := vm["vmid"].(float64); ok {
			if int(vmid) > highest {
				highest = int(vmid)
			}
		}
	}

	return highest + 1, nil
}
