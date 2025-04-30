package handlers

import (
	"encoding/json"
	"fmt"
	"rm-thierry/Proxmox-API/src/manager"
	"strings"
)

type ContainerConfig struct {
	Node         string `json:"node"`
	CTID         string `json:"ctid"`
	Name         string `json:"name"`
	Memory       string `json:"memory"`
	Swap         string `json:"swap"`
	Cores        string `json:"cores"`
	Disk         string `json:"disk"`
	Storage      string `json:"storage"`
	Net          string `json:"net"`
	Password     string `json:"password"`
	Template     string `json:"template"`
	Unprivileged bool   `json:"unprivileged"`
}

type Template struct {
	Debian string `json:"debian"`
	Ubuntu string `json:"ubuntu"`
	Alpine string `json:"alpine"`
}

func GetTemplates() Template {
	return Template{
		Debian: "local:vztmpl/debian-12-standard_12.7-1_amd64.tar.zst",
	}
}

func NewDefaultContainerConfig() ContainerConfig {
	return ContainerConfig{
		Node:         manager.NewAPIManager().Node,
		Memory:       "2000",
		Swap:         "2000",
		Cores:        "2",
		Disk:         "8",
		Storage:      "local",
		Net:          "name=eth0,bridge=vmbr0,ip=dhcp",
		Template:     GetTemplates().Debian,
		Unprivileged: true,
		Password:     "",
	}
}

func validateContainer(apiManager *manager.APIManager, config ContainerConfig) error {
	if config.CTID == "" || config.Name == "" {
		return fmt.Errorf("CTID and Name are required")
	}

	if config.Password == "" {
		return fmt.Errorf("root password is required")
	}

	exists, err := checkContainerExists(apiManager, config.Node, config.CTID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("Container with ID %s already exists", config.CTID)
	}

	if err := validateStorage(apiManager, VMConfig{Node: config.Node, Disk: config.Storage}); err != nil {
		return err
	}

	if config.Disk == "" {
		return fmt.Errorf("disk size is required")
	}

	return validateContainerTemplate(apiManager, config)
}

func checkContainerExists(apiManager *manager.APIManager, node, ctid string) (bool, error) {
	containers, err := GetContainers(apiManager, node)
	if err != nil {
		return false, err
	}

	for _, container := range containers {
		if id, ok := container["vmid"].(float64); ok {
			if fmt.Sprintf("%.0f", id) == ctid {
				return true, nil
			}
		}
	}
	return false, nil
}

func validateContainerTemplate(apiManager *manager.APIManager, config ContainerConfig) error {
	response, err := apiManager.ApiCall("GET", fmt.Sprintf("/nodes/%s/storage/%s/content", config.Node, config.Storage), nil)
	if err != nil {
		return fmt.Errorf("failed to check template: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return err
	}

	data, ok := result["data"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid template response")
	}

	templateName := strings.Split(config.Template, "/")[1]
	for _, t := range data {
		if template, ok := t.(map[string]interface{}); ok {
			if volid, ok := template["volid"].(string); ok && strings.Contains(volid, templateName) {
				return nil
			}
		}
	}
	return fmt.Errorf("template %s not found", config.Template)
}

func buildContainerPayload(config ContainerConfig) map[string]interface{} {
	return map[string]interface{}{
		"vmid":         config.CTID,
		"hostname":     config.Name,
		"cores":        config.Cores,
		"memory":       config.Memory,
		"swap":         config.Swap,
		"storage":      config.Storage,
		"rootfs":       fmt.Sprintf("%s:%s", config.Storage, config.Disk),
		"net0":         config.Net,
		"ostemplate":   config.Template,
		"unprivileged": config.Unprivileged,
		"password":     config.Password,
	}
}

func GetContainers(apiManager *manager.APIManager, node string) ([]map[string]interface{}, error) {
	response, err := apiManager.ApiCall("GET", fmt.Sprintf("/nodes/%s/lxc", node), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get containers: %v", err)
	}

	result, err := parseAPIResponse(response)
	if err != nil {
		return nil, err
	}

	data, ok := result["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	containers := make([]map[string]interface{}, len(data))
	for i, item := range data {
		containers[i] = item.(map[string]interface{})
	}

	return containers, nil
}

func GetContainer(apiManager *manager.APIManager, node string, ctid string) (map[string]interface{}, error) {
	response, err := apiManager.ApiCall("GET", fmt.Sprintf("/nodes/%s/lxc/%s/status/current", node, ctid), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get container: %v", err)
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

func GetContainerIDByName(apiManager *manager.APIManager, node string, name string) (string, error) {
	containers, err := GetContainers(apiManager, node)
	if err != nil {
		return "", err
	}

	for _, container := range containers {
		if cname, ok := container["name"].(string); ok && cname == name {
			if ctid, ok := container["vmid"].(float64); ok {
				return fmt.Sprintf("%.0f", ctid), nil
			}
		}
	}

	return "", fmt.Errorf("Container not found")
}

func CreateContainer(apiManager *manager.APIManager, config ContainerConfig) (map[string]interface{}, error) {
	if err := validateContainer(apiManager, config); err != nil {
		return nil, err
	}

	payload := buildContainerPayload(config)
	response, err := apiManager.ApiCall("POST", fmt.Sprintf("/nodes/%s/lxc", config.Node), payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %v", err)
	}
	return parseAPIResponse(response)
}

func DeleteContainer(apiManager *manager.APIManager, node string, ctid string) (map[string]interface{}, error) {
	exists, err := checkContainerExists(apiManager, node, ctid)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("Container with ID %s does not exist", ctid)
	}

	response, err := apiManager.ApiCall("DELETE", fmt.Sprintf("/nodes/%s/lxc/%s", node, ctid), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to delete container: %v", err)
	}

	return parseAPIResponse(response)
}

func StartContainer(apiManager *manager.APIManager, node string, ctid string) (map[string]interface{}, error) {
	exists, err := checkContainerExists(apiManager, node, ctid)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("Container with ID %s does not exist", ctid)
	}

	response, err := apiManager.ApiCall("POST", fmt.Sprintf("/nodes/%s/lxc/%s/status/start", node, ctid), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %v", err)
	}

	return parseAPIResponse(response)
}

func StopContainer(apiManager *manager.APIManager, node string, ctid string) (map[string]interface{}, error) {
	exists, err := checkContainerExists(apiManager, node, ctid)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("Container with ID %s does not exist", ctid)
	}

	response, err := apiManager.ApiCall("POST", fmt.Sprintf("/nodes/%s/lxc/%s/status/stop", node, ctid), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to stop container: %v", err)
	}

	return parseAPIResponse(response)
}

func GetHighestContainerID(apiManager *manager.APIManager, node string) (int, error) {
	response, err := apiManager.ApiCall("GET", fmt.Sprintf("/nodes/%s/lxc", node), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get containers: %w", err)
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
		container, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		if ctid, ok := container["vmid"].(float64); ok {
			if int(ctid) > highest {
				highest = int(ctid)
			}
		}
	}

	return highest + 1, nil
}