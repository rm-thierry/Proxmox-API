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
	Debian  string   `json:"debian"`
	Ubuntu  string   `json:"ubuntu"`
	Windows string   `json:"windows"`
	All     []string `json:"all"`
}

func GetISOs(apiManager *manager.APIManager, node string) ISO {
	defaultISO := ISO{
		Debian:  "local:iso/debian-12.9.0-amd64-netinst.iso",
		Ubuntu:  "local:iso/ubuntu-22.04.3-live-server-amd64.iso",
		Windows: "local:iso/windows-server-2022.iso",
		All:     []string{},
	}
	
	// Try to get available ISOs from API
	availableISOs, err := getAvailableISOs(apiManager, node)
	if err != nil {
		// Return default ISOs if API call fails
		return defaultISO
	}
	
	// Set All field with available ISOs
	defaultISO.All = availableISOs
	
	// Update specific OS ISOs if they exist in available ISOs
	for _, iso := range availableISOs {
		if strings.Contains(strings.ToLower(iso), "debian") {
			defaultISO.Debian = iso
		} else if strings.Contains(strings.ToLower(iso), "ubuntu") {
			defaultISO.Ubuntu = iso
		} else if strings.Contains(strings.ToLower(iso), "windows") {
			defaultISO.Windows = iso
		}
	}
	
	return defaultISO
}

func NewDefaultVMConfig() VMConfig {
	apiManager := manager.NewAPIManager()
	
	// Try to get available storages to set a valid default
	defaultStorage := "local"
	
	// First try to get storages from cluster level (more comprehensive)
	clusterResponse, err := apiManager.ApiCall("GET", "/cluster/resources?type=storage", nil)
	if err == nil {
		var clusterResult map[string]interface{}
		if err := json.Unmarshal(clusterResponse, &clusterResult); err == nil {
			if clusterData, ok := clusterResult["data"].([]interface{}); ok && len(clusterData) > 0 {
				// Use the first available storage from cluster level
				if store, ok := clusterData[0].(map[string]interface{}); ok {
					if storageVal, ok := store["storage"].(string); ok {
						defaultStorage = storageVal
					}
				}
			}
		}
	}
	
	// Fallback to node-level storages if needed
	if defaultStorage == "local" {
		storages, err := getAvailableStorages(apiManager, apiManager.Node)
		if err == nil && len(storages) > 0 {
			// Use the first available storage
			if name, ok := storages[0]["name"].(string); ok {
				defaultStorage = name
			}
		}
	}
	
	return VMConfig{
		Node:    apiManager.Node,
		Cores:   "2",
		Memory:  "4000",
		Disk:    defaultStorage,
		Net:     "vmbr0",
		ISO:     GetISOs(apiManager, apiManager.Node).Debian,
		OSType:  "l26",
		CPU:     "host",
		Sockets: "1",
	}
}

func getAvailableStorages(apiManager *manager.APIManager, node string) ([]map[string]interface{}, error) {
	// First, try to get storages from cluster resources (more comprehensive)
	clusterStorages := make([]map[string]interface{}, 0)
	
	clusterResponse, err := apiManager.ApiCall("GET", "/cluster/resources?type=storage", nil)
	if err == nil {
		var clusterResult map[string]interface{}
		if err := json.Unmarshal(clusterResponse, &clusterResult); err == nil {
			if clusterData, ok := clusterResult["data"].([]interface{}); ok {
				for _, s := range clusterData {
					if store, ok := s.(map[string]interface{}); ok {
						storageVal, ok := store["storage"].(string)
						if !ok {
							continue
						}
						
						// Get storage details
						storageDetail := map[string]interface{}{
							"name": storageVal,
							"type": store["type"],
						}
						
						// Try to get storage content types
						if content, ok := store["content"].(string); ok {
							contentTypes := strings.Split(content, ",")
							storageDetail["contentTypes"] = contentTypes
						}
						
						clusterStorages = append(clusterStorages, storageDetail)
					}
				}
			}
		}
	}
	
	// If we got storages from the cluster, return them
	if len(clusterStorages) > 0 {
		return clusterStorages, nil
	}
	
	// Fall back to node-specific storage list
	storageResponse, err := apiManager.ApiCall("GET", fmt.Sprintf("/nodes/%s/storage", node), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get storages: %v", err)
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(storageResponse, &result); err != nil {
		return nil, err
	}
	
	data, ok := result["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid storage response")
	}
	
	storages := make([]map[string]interface{}, 0)
	for _, s := range data {
		if store, ok := s.(map[string]interface{}); ok {
			storageVal, ok := store["storage"].(string)
			if !ok {
				continue
			}
			
			// Get storage details
			storageDetail := map[string]interface{}{
				"name": storageVal,
				"type": store["type"],
			}
			
			// Try to get storage content types
			if content, ok := store["content"].(string); ok {
				contentTypes := strings.Split(content, ",")
				storageDetail["contentTypes"] = contentTypes
			}
			
			storages = append(storages, storageDetail)
		}
	}
	
	return storages, nil
}

func GetAvailableResources(apiManager *manager.APIManager, node string) (map[string]interface{}, error) {
	resources := make(map[string]interface{})
	
	// Get available storages with details
	storages, err := getAvailableStorages(apiManager, node)
	if err == nil {
		resources["storages"] = storages
		
		// Also provide a simple list of storage names for backward compatibility
		storageNames := make([]string, len(storages))
		for i, storage := range storages {
			storageNames[i] = storage["name"].(string)
		}
		resources["storageNames"] = storageNames
	}
	
	// Get available networks
	networkResponse, err := apiManager.ApiCall("GET", fmt.Sprintf("/nodes/%s/network", node), nil)
	if err == nil {
		var result map[string]interface{}
		if err := json.Unmarshal(networkResponse, &result); err == nil {
			if data, ok := result["data"].([]interface{}); ok {
				networks := []string{}
				networkDetails := make([]map[string]interface{}, 0)
				
				for _, n := range data {
					if net, ok := n.(map[string]interface{}); ok {
						if ifaceVal, ok := net["iface"].(string); ok {
							networks = append(networks, ifaceVal)
							
							// Get network details
							netDetail := map[string]interface{}{
								"name": ifaceVal,
								"type": net["type"],
							}
							
							// Add additional network information if available
							if active, ok := net["active"].(bool); ok {
								netDetail["active"] = active
							}
							
							networkDetails = append(networkDetails, netDetail)
						}
					}
				}
				resources["networks"] = networkDetails
				resources["networkNames"] = networks
			}
		}
	}
	
	// Get available ISOs
	isos, err := getAvailableISOs(apiManager, node)
	if err == nil {
		resources["isos"] = isos
	}
	
	// Get detailed ISO information
	isoDetails, err := getAvailableISODetails(apiManager, node)
	if err == nil {
		resources["isoDetails"] = isoDetails
	}
	
	return resources, nil
}

func getAvailableISOs(apiManager *manager.APIManager, node string) ([]string, error) {
	isoDetails, err := getAvailableISODetails(apiManager, node)
	if err != nil {
		return nil, err
	}
	
	isos := make([]string, len(isoDetails))
	for i, iso := range isoDetails {
		isos[i] = iso["volid"].(string)
	}
	
	return isos, nil
}

func getAvailableISODetails(apiManager *manager.APIManager, node string) ([]map[string]interface{}, error) {
	// Try to get ISOs from all storages that can contain ISOs
	storages, err := getAvailableStorages(apiManager, node)
	if err != nil {
		return nil, fmt.Errorf("failed to get storages: %v", err)
	}
	
	// If no storages found, try to get from cluster level
	if len(storages) == 0 {
		clusterResponse, err := apiManager.ApiCall("GET", "/cluster/resources?type=storage", nil)
		if err == nil {
			var clusterResult map[string]interface{}
			if err := json.Unmarshal(clusterResponse, &clusterResult); err == nil {
				if clusterData, ok := clusterResult["data"].([]interface{}); ok {
					for _, s := range clusterData {
						if store, ok := s.(map[string]interface{}); ok {
							storageVal, ok := store["storage"].(string)
							if !ok {
								continue
							}
							
							// Get storage details
							storageDetail := map[string]interface{}{
								"name": storageVal,
								"type": store["type"],
							}
							
							// Try to get storage content types
							if content, ok := store["content"].(string); ok {
								contentTypes := strings.Split(content, ",")
								storageDetail["contentTypes"] = contentTypes
							}
							
							storages = append(storages, storageDetail)
						}
					}
				}
			}
		}
	}
	
	isoDetails := []map[string]interface{}{}
	
	// Loop through all storages
	for _, storage := range storages {
		storageName, ok := storage["name"].(string)
		if !ok {
			continue
		}
		
		// Check if the storage supports ISO content
		if contentTypes, ok := storage["contentTypes"].([]string); ok {
			supportsISO := false
			for _, contentType := range contentTypes {
				if contentType == "iso" {
					supportsISO = true
					break
				}
			}
			
			if !supportsISO {
				continue
			}
		}
		
		// Get ISOs from this storage
		response, err := apiManager.ApiCall("GET", fmt.Sprintf("/nodes/%s/storage/%s/content", node, storageName), nil)
		if err != nil {
			// Skip this storage if there's an error
			continue
		}
		
		var result map[string]interface{}
		if err := json.Unmarshal(response, &result); err != nil {
			// Skip this storage if there's an error
			continue
		}
		
		data, ok := result["data"].([]interface{})
		if !ok {
			// Skip this storage if the response format is invalid
			continue
		}
		
		for _, item := range data {
			content, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			
			contentType, ok := content["content"].(string)
			if !ok || contentType != "iso" {
				continue
			}
			
			volid, ok := content["volid"].(string)
			if !ok {
				continue
			}
			
			// Create ISO detail object
			isoDetail := map[string]interface{}{
				"volid": volid,
				"storage": storageName,
			}
			
			// Add additional ISO information if available
			if format, ok := content["format"].(string); ok {
				isoDetail["format"] = format
			}
			if size, ok := content["size"].(float64); ok {
				isoDetail["size"] = size
			}
			
			isoDetails = append(isoDetails, isoDetail)
		}
	}
	
	return isoDetails, nil
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

	// Validate ISO if specified
	if config.ISO != "" {
		isos, err := getAvailableISOs(apiManager, config.Node)
		if err != nil {
			return fmt.Errorf("failed to check ISO: %v", err)
		}

		found := false
		for _, iso := range isos {
			if iso == config.ISO {
				found = true
				break
			}
		}

		if !found {
			availableISOsStr := strings.Join(isos, ", ")
			return fmt.Errorf("ISO %s not found. Available ISOs: [%s]. Use GET /api/v1/resources for complete resource information", config.ISO, availableISOsStr)
		}
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
	// Extract storage name from disk parameter
	// Handle both "storage" and "storage:size" formats
	storage := config.Disk
	if parts := strings.Split(config.Disk, ":"); len(parts) > 0 {
		storage = parts[0]
	}
	
	// First, try cluster-level resources which gives a more complete view of all storages
	clusterResponse, err := apiManager.ApiCall("GET", "/cluster/resources?type=storage", nil)
	if err == nil {
		var clusterResult map[string]interface{}
		if err := json.Unmarshal(clusterResponse, &clusterResult); err == nil {
			clusterData, ok := clusterResult["data"].([]interface{})
			if ok && len(clusterData) > 0 {
				// Collect all available storages from cluster
				availableStorages := []string{}
				for _, s := range clusterData {
					if store, ok := s.(map[string]interface{}); ok {
						if storageVal, ok := store["storage"].(string); ok {
							availableStorages = append(availableStorages, storageVal)
							if storageVal == storage {
								return nil
							}
						}
					}
				}
				
				// If we have storages but didn't find a match, return the error
				if len(availableStorages) > 0 {
					availableStoragesStr := strings.Join(availableStorages, ", ")
					return fmt.Errorf("storage %s not found. Available storages: [%s]. Use GET /api/v1/resources for complete resource information", 
						storage, availableStoragesStr)
				}
			}
		}
	}
	
	// Fall back to node-specific storage list if cluster approach didn't work
	response, err := apiManager.ApiCall("GET", fmt.Sprintf("/nodes/%s/storage", config.Node), nil)
	if err != nil {
		return fmt.Errorf("failed to check storage on node %s: %v", config.Node, err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("failed to parse storage response: %v", err)
	}

	data, ok := result["data"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid storage response format")
	}

	// Collect all available storages for error message
	availableStorages := []string{}
	for _, s := range data {
		if store, ok := s.(map[string]interface{}); ok {
			if storageVal, ok := store["storage"].(string); ok {
				availableStorages = append(availableStorages, storageVal)
				if storageVal == storage {
					return nil
				}
			}
		}
	}
	
	// Format available storages list for error message
	availableStoragesStr := strings.Join(availableStorages, ", ")
	return fmt.Errorf("storage %s not found. Available storages: [%s]. Use GET /api/v1/resources for complete resource information", 
		storage, availableStoragesStr)
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

	// Collect all available network bridges for error message
	availableNetworks := []string{}
	for _, n := range data {
		if net, ok := n.(map[string]interface{}); ok {
			if ifaceVal, ok := net["iface"].(string); ok {
				availableNetworks = append(availableNetworks, ifaceVal)
				if ifaceVal == config.Net {
					return nil
				}
			}
		}
	}
	
	// Default vmbr0 is always allowed even if it's not in the API response
	if config.Net == "vmbr0" {
		return nil
	}
	
	// Format available networks list for error message
	availableNetworksStr := strings.Join(availableNetworks, ", ")
	return fmt.Errorf("network bridge %s not found. Available bridges: [%s]. Use GET /api/v1/resources for complete resource information", config.Net, availableNetworksStr)
}

func buildVMPayload(config VMConfig) map[string]interface{} {
	// Handle disk configuration
	diskConfig := config.Disk
	// If disk doesn't already include a size specification (no colon), add a default ":0" 
	// which tells Proxmox to use the default size
	if !strings.Contains(diskConfig, ":") {
		diskConfig = diskConfig + ":0"
	}
	
	payload := map[string]interface{}{
		"vmid":     config.VMID,
		"name":     config.Name,
		"cores":    config.Cores,
		"memory":   config.Memory,
		"virtio0":  diskConfig,
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
