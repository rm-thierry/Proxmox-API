package handlers

import (
	"encoding/json"
	"fmt"
	"rm-thierry/Proxmox-API/src/manager"
	"strconv"
	"strings"
)

type VMCreateRequest struct {
	Node    string `json:"node"`
	VMID    string `json:"vmid"`
	Name    string `json:"name"`
	Cores   int    `json:"cores"`
	Memory  int    `json:"memory"`
	Disk    string `json:"disk"`
	Net     string `json:"net"`
	ISO     string `json:"iso"`
	OSType  string `json:"ostype"`
	CPU     string `json:"cpu"`
	Sockets int    `json:"sockets"`
}

type VMConfig struct {
	VMCreateRequest
}

func ListVMs(api *manager.APIManager, node string) ([]map[string]interface{}, error) {
	response, err := api.ApiCall("GET", fmt.Sprintf("/nodes/%s/qemu", node), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse VM list: %w", err)
	}

	data, ok := result["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid VM list format")
	}

	vms := make([]map[string]interface{}, len(data))
	for i, vm := range data {
		vms[i] = vm.(map[string]interface{})
	}

	return vms, nil
}

func CreateVM(api *manager.APIManager, req *VMCreateRequest) (map[string]interface{}, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("VM name is required")
	}
	if req.Cores <= 0 {
		return nil, fmt.Errorf("cores must be greater than 0")
	}
	if req.Memory <= 0 {
		return nil, fmt.Errorf("memory must be greater than 0")
	}
	if !strings.Contains(req.Disk, ":") {
		return nil, fmt.Errorf("disk must be in format 'storage:sizeG'")
	}
	if req.Net == "" {
		return nil, fmt.Errorf("network bridge is required")
	}
	if req.ISO == "" {
		return nil, fmt.Errorf("ISO is required")
	}

	if req.VMID == "" {
		vmid, err := generateVMID(api, req.Node)
		if err != nil {
			return nil, fmt.Errorf("failed to generate VMID: %w", err)
		}
		req.VMID = strconv.Itoa(vmid)
	}

	if _, err := strconv.Atoi(req.VMID); err != nil {
		return nil, fmt.Errorf("VMID must be a number")
	}

	if exists, _ := VMExists(api, req.Node, req.VMID); exists {
		return nil, fmt.Errorf("VM with ID %s already exists", req.VMID)
	}

	if err := validateResources(api, req); err != nil {
		return nil, err
	}

	payload := buildVMPayload(req)
	response, err := api.ApiCall("POST", fmt.Sprintf("/nodes/%s/qemu", req.Node), payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}

	return parseResponse(response)
}

func GetVM(api *manager.APIManager, node, vmid string) (map[string]interface{}, error) {
	if _, err := strconv.Atoi(vmid); err != nil {
		return nil, fmt.Errorf("invalid VMID format")
	}

	response, err := api.ApiCall("GET", fmt.Sprintf("/nodes/%s/qemu/%s/status/current", node, vmid), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM: %w", err)
	}

	return parseResponse(response)
}

func DeleteVM(api *manager.APIManager, node, vmid string) error {
	if _, err := strconv.Atoi(vmid); err != nil {
		return fmt.Errorf("invalid VMID format")
	}

	if exists, _ := VMExists(api, node, vmid); !exists {
		return fmt.Errorf("VM with ID %s not found", vmid)
	}

	_, err := api.ApiCall("DELETE", fmt.Sprintf("/nodes/%s/qemu/%s", node, vmid), nil)
	if err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}

	return nil
}

func StartVM(api *manager.APIManager, node, vmid string) error {
	return vmOperation(api, node, vmid, "start")
}

func StopVM(api *manager.APIManager, node, vmid string) error {
	return vmOperation(api, node, vmid, "stop")
}

func RebootVM(api *manager.APIManager, node, vmid string) error {
	return vmOperation(api, node, vmid, "reboot")
}

func vmOperation(api *manager.APIManager, node, vmid, operation string) error {
	if _, err := strconv.Atoi(vmid); err != nil {
		return fmt.Errorf("invalid VMID format")
	}

	if exists, _ := VMExists(api, node, vmid); !exists {
		return fmt.Errorf("VM with ID %s not found", vmid)
	}

	_, err := api.ApiCall("POST", fmt.Sprintf("/nodes/%s/qemu/%s/status/%s", node, vmid, operation), nil)
	if err != nil {
		return fmt.Errorf("failed to %s VM: %w", operation, err)
	}

	return nil
}

func VMExists(api *manager.APIManager, node, vmid string) (bool, error) {
	vms, err := ListVMs(api, node)
	if err != nil {
		return false, err
	}

	for _, vm := range vms {
		if id, ok := vm["vmid"].(float64); ok && fmt.Sprintf("%.0f", id) == vmid {
			return true, nil
		}
	}

	return false, nil
}

func GetResources(api *manager.APIManager, node string) (map[string]interface{}, error) {
	resources := make(map[string]interface{})

	storages, err := GetStorages(api, node)
	if err == nil {
		resources["storages"] = storages
	}

	networks, err := GetNetworks(api, node)
	if err == nil {
		resources["networks"] = networks
	}

	isos, err := GetISOs(api, node)
	if err == nil {
		resources["isos"] = isos
	}

	return resources, nil
}

func GetStorages(api *manager.APIManager, node string) ([]map[string]interface{}, error) {
	response, err := api.ApiCall("GET", fmt.Sprintf("/nodes/%s/storage", node), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get storages: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse storages: %w", err)
	}

	data, ok := result["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid storages format")
	}

	storages := make([]map[string]interface{}, len(data))
	for i, storage := range data {
		storages[i] = storage.(map[string]interface{})
	}

	return storages, nil
}

func GetNetworks(api *manager.APIManager, node string) ([]map[string]interface{}, error) {
	response, err := api.ApiCall("GET", fmt.Sprintf("/nodes/%s/network", node), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get networks: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse networks: %w", err)
	}

	data, ok := result["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid networks format")
	}

	networks := make([]map[string]interface{}, len(data))
	for i, network := range data {
		networks[i] = network.(map[string]interface{})
	}

	return networks, nil
}

func GetISOs(api *manager.APIManager, node string) ([]map[string]interface{}, error) {
	storages, err := GetStorages(api, node)
	if err != nil {
		return nil, fmt.Errorf("failed to get storages: %w", err)
	}

	var isos []map[string]interface{}
	for _, storage := range storages {
		storageName, ok := storage["storage"].(string)
		if !ok {
			continue
		}

		if content, ok := storage["content"].(string); ok && strings.Contains(content, "iso") {
			response, err := api.ApiCall("GET",
				fmt.Sprintf("/nodes/%s/storage/%s/content", node, storageName), nil)
			if err != nil {
				continue
			}

			var result map[string]interface{}
			if err := json.Unmarshal(response, &result); err != nil {
				continue
			}

			data, ok := result["data"].([]interface{})
			if !ok {
				continue
			}

			for _, item := range data {
				content, ok := item.(map[string]interface{})
				if !ok {
					continue
				}

				if contentType, ok := content["content"].(string); ok && contentType == "iso" {
					isos = append(isos, content)
				}
			}
		}
	}

	return isos, nil
}

func generateVMID(api *manager.APIManager, node string) (int, error) {
	vms, err := ListVMs(api, node)
	if err != nil {
		return 0, err
	}

	highest := 100
	for _, vm := range vms {
		if vmid, ok := vm["vmid"].(float64); ok && int(vmid) > highest {
			highest = int(vmid)
		}
	}

	return highest + 1, nil
}

func validateResources(api *manager.APIManager, req *VMCreateRequest) error {
	storageParts := strings.Split(req.Disk, ":")
	if len(storageParts) != 2 {
		return fmt.Errorf("invalid disk format")
	}

	storages, err := GetStorages(api, req.Node)
	if err != nil {
		return fmt.Errorf("failed to validate storage: %w", err)
	}

	storageValid := false
	for _, storage := range storages {
		if name, ok := storage["storage"].(string); ok && name == storageParts[0] {
			storageValid = true
			break
		}
	}

	if !storageValid {
		return fmt.Errorf("storage '%s' not found", storageParts[0])
	}

	networks, err := GetNetworks(api, req.Node)
	if err != nil {
		return fmt.Errorf("failed to validate network: %w", err)
	}

	networkValid := false
	for _, network := range networks {
		if iface, ok := network["iface"].(string); ok && iface == req.Net {
			networkValid = true
			break
		}
	}

	if !networkValid && !strings.HasPrefix(req.Net, "vmbr") {
		return fmt.Errorf("network '%s' not found", req.Net)
	}

	isos, err := GetISOs(api, req.Node)
	if err != nil {
		return fmt.Errorf("failed to validate ISO: %w", err)
	}

	isoValid := false
	for _, iso := range isos {
		if volid, ok := iso["volid"].(string); ok && volid == req.ISO {
			isoValid = true
			break
		}
	}

	if !isoValid && !strings.HasPrefix(req.ISO, "local:iso/") {
		return fmt.Errorf("ISO '%s' not found", req.ISO)
	}

	return nil
}

func buildVMPayload(req *VMCreateRequest) map[string]interface{} {
	diskParts := strings.Split(req.Disk, ":")
	storage := diskParts[0]
	size := strings.TrimSuffix(diskParts[1], "G")

	payload := map[string]interface{}{
		"vmid":     req.VMID,
		"name":     req.Name,
		"cores":    req.Cores,
		"memory":   req.Memory,
		"sockets":  req.Sockets,
		"cpu":      req.CPU,
		"ostype":   req.OSType,
		"virtio0":  fmt.Sprintf("%s:%s,format=raw", storage, size),
		"net0":     fmt.Sprintf("virtio,bridge=%s", req.Net),
		"ide2":     fmt.Sprintf("%s,media=cdrom", req.ISO),
		"scsihw":   "virtio-scsi-pci",
		"bootdisk": "virtio0",
		"acpi":     1,
	}

	return payload
}

func parseResponse(response []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		return data, nil
	}

	if dataStr, ok := result["data"].(string); ok {
		return map[string]interface{}{
			"task_id": dataStr,
		}, nil
	}

	return nil, fmt.Errorf("invalid response format")
}
