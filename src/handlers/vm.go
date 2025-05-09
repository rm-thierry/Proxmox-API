package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"rm-thierry/Proxmox-API/src/manager"
	"strconv"
	"strings"
	"time"
)

type VMCreateRequest struct {
	Node         string `json:"node"`
	VMID         string `json:"vmid"`
	Name         string `json:"name"`
	Cores        int    `json:"cores"`
	Memory       int    `json:"memory"`
	Disk         string `json:"disk"`
	Net          string `json:"net"`
	ISO          string `json:"iso"`
	OSType       string `json:"ostype"`
	CPU          string `json:"cpu"`
	Sockets      int    `json:"sockets"`
	Template     string `json:"template,omitempty"`
	CloudInit    bool   `json:"cloudinit,omitempty"`
	SSHKeys      string `json:"sshkeys,omitempty"`
	Nameserver   string `json:"nameserver,omitempty"`
	Searchdomain string `json:"searchdomain,omitempty"`
	Ciuser       string `json:"ciuser,omitempty"`
	Cipassword   string `json:"cipassword,omitempty"`
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

func CloneVM(api *manager.APIManager, sourceNode string, sourceVMID string, targetNode string, targetVMID string, name string) (map[string]interface{}, error) {
	if sourceVMID == "" {
		return nil, fmt.Errorf("source VMID is required")
	}

	if _, err := strconv.Atoi(sourceVMID); err != nil {
		return nil, fmt.Errorf("source VMID must be a number")
	}

	if targetVMID == "" {
		vmid, err := generateVMID(api, targetNode)
		if err != nil {
			return nil, fmt.Errorf("failed to generate target VMID: %w", err)
		}
		targetVMID = strconv.Itoa(vmid)
	}

	if _, err := strconv.Atoi(targetVMID); err != nil {
		return nil, fmt.Errorf("target VMID must be a number")
	}

	if exists, _ := VMExists(api, targetNode, targetVMID); exists {
		return nil, fmt.Errorf("VM with ID %s already exists", targetVMID)
	}

	// Prepare clone payload
	payload := map[string]interface{}{
		"newid":  targetVMID,
		"full":   1, // Full clone (not linked)
		"target": targetNode,
	}

	if name != "" {
		payload["name"] = name
	}

	// Execute the clone operation
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%s/clone", sourceNode, sourceVMID)
	response, err := api.ApiCall("POST", endpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to clone VM: %w", err)
	}

	return parseResponse(response)
}

func CreateVM(api *manager.APIManager, req *VMCreateRequest) (map[string]interface{}, error) {
	// Check if we should use the template approach
	if req.Template != "" {
		return CreateVMFromTemplate(api, req)
	}

	// Validate required fields for standard VM creation
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
	// ISO is required only when not using cloud-init
	if !req.CloudInit && req.ISO == "" {
		return nil, fmt.Errorf("ISO is required when not using cloud-init")
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

func CreateVMFromTemplate(api *manager.APIManager, req *VMCreateRequest) (map[string]interface{}, error) {
	// Get template configurations
	templates, err := GetVMTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Check if the requested template exists
	templateVMID, ok := templates[req.Template]
	if !ok {
		return nil, fmt.Errorf("template '%s' not found", req.Template)
	}

	// Generate a VMID if not provided
	if req.VMID == "" {
		vmid, err := generateVMID(api, req.Node)
		if err != nil {
			return nil, fmt.Errorf("failed to generate VMID: %w", err)
		}
		req.VMID = strconv.Itoa(vmid)
	}

	// Default the name if not provided
	if req.Name == "" {
		req.Name = fmt.Sprintf("%s-vm-%s", req.Template, req.VMID)
	}

	// Clone the template VM
	// We assume the template is on the same node for simplicity
	result, err := CloneVM(api, req.Node, templateVMID, req.Node, req.VMID, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to clone template VM: %w", err)
	}

	// Wait for clone to complete
	// This is a simple approach - in production you might want to poll the task status
	time.Sleep(3 * time.Second)

	// Update VM configuration based on request
	updatePayload := make(map[string]interface{})

	// Apply core count if specified
	if req.Cores > 0 {
		updatePayload["cores"] = req.Cores
	}

	// Apply memory if specified
	if req.Memory > 0 {
		updatePayload["memory"] = req.Memory
	}

	// Apply CPU type if specified
	if req.CPU != "" {
		updatePayload["cpu"] = req.CPU
	}

	// Apply sockets if specified
	if req.Sockets > 0 {
		updatePayload["sockets"] = req.Sockets
	}

	// Apply net0 if specified
	if req.Net != "" {
		updatePayload["net0"] = fmt.Sprintf("virtio,bridge=%s", req.Net)
	}

	// Apply disk size if specified
	if req.Disk != "" && strings.Contains(req.Disk, ":") {
		diskParts := strings.Split(req.Disk, ":")
		storage := diskParts[0]
		size := strings.TrimSuffix(diskParts[1], "G")

		updatePayload["virtio0"] = fmt.Sprintf("%s:%s,format=raw", storage, size)
	}

	// For CloudInit, only set the drive if it doesn't already exist
	// We'll check the current config first to avoid the "already exists" error
	vmConfig, err := GetVMConfig(api, req.Node, req.VMID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM config: %w", err)
	}

	// Only set the CloudInit drive if it's not already configured
	// This prevents the "Logical Volume already exists" error
	if _, hasCloudInit := vmConfig["ide2"]; !hasCloudInit {
		updatePayload["ide2"] = "local-lvm:cloudinit"
	}

	// Always set DHCP for CloudInit
	updatePayload["ipconfig0"] = "ip=dhcp"

	if req.SSHKeys != "" {
		// Proxmox expects the SSH keys to be properly formatted with newlines
		updatePayload["sshkeys"] = strings.ReplaceAll(req.SSHKeys, " ", "\n")
	}

	if req.Nameserver != "" {
		updatePayload["nameserver"] = req.Nameserver
	}

	if req.Searchdomain != "" {
		updatePayload["searchdomain"] = req.Searchdomain
	}

	// Set CloudInit user if provided
	if req.Ciuser != "" {
		updatePayload["ciuser"] = req.Ciuser
	}

	// Set CloudInit password if provided
	if req.Cipassword != "" {
		updatePayload["cipassword"] = req.Cipassword
	}

	// Apply the configuration updates
	if len(updatePayload) > 0 {
		_, err = api.ApiCall("POST", fmt.Sprintf("/nodes/%s/qemu/%s/config", req.Node, req.VMID), updatePayload)
		if err != nil {
			return nil, fmt.Errorf("VM cloned but failed to update configuration: %w", err)
		}
	}

	// Start the VM if CloudInit is configured
	if req.CloudInit {
		err = StartVM(api, req.Node, req.VMID)
		if err != nil {
			return result, fmt.Errorf("VM cloned and configured but failed to start: %w", err)
		}
	}

	return result, nil
}

func GetVMTemplates() (map[string]string, error) {
	// Load templates from the JSON file
	file, err := os.ReadFile("env/templates.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read templates file: %w", err)
	}

	var templateData struct {
		Templates map[string]string `json:"templates"`
	}

	if err := json.Unmarshal(file, &templateData); err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return templateData.Templates, nil
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
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%s/status/start", node, vmid)
	_, err := api.ApiCallWithOptions("POST", endpoint, nil, false)
	if err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}
	return nil
}

func StopVM(api *manager.APIManager, node, vmid string) error {
	return vmOperation(api, node, vmid, "stop")
}

func RebootVM(api *manager.APIManager, node, vmid string) error {
	return vmOperation(api, node, vmid, "reboot")
}

func vmOperation(api *manager.APIManager, node, vmid, operation string) error {
	// Validate VMID
	if _, err := strconv.Atoi(vmid); err != nil {
		return fmt.Errorf("invalid VMID format")
	}

	// Check if VM exists
	if exists, _ := VMExists(api, node, vmid); !exists {
		return fmt.Errorf("VM with ID %s not found", vmid)
	}

	// Send API request to perform the operation without JSON content type
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%s/status/%s", node, vmid, operation)
	_, err := api.ApiCallWithOptions("POST", endpoint, nil, false)
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

func GetNodes(api *manager.APIManager) ([]map[string]interface{}, error) {
	response, err := api.ApiCall("GET", "/nodes", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse nodes: %w", err)
	}

	data, ok := result["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid nodes format")
	}

	nodes := make([]map[string]interface{}, len(data))
	for i, node := range data {
		nodes[i] = node.(map[string]interface{})
	}

	return nodes, nil
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

	usedVMIDs := make(map[int]bool)
	for _, vm := range vms {
		if vmid, ok := vm["vmid"].(float64); ok {
			usedVMIDs[int(vmid)] = true
		}
	}

	for vmid := 2000; vmid <= 3000; vmid++ {
		if !usedVMIDs[vmid] {
			return vmid, nil
		}
	}

	return 0, fmt.Errorf("no available VMID in the range 2000-3000")
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
		"scsihw":   "virtio-scsi-pci",
		"bootdisk": "virtio0",
		"acpi":     1,
	}

	// Handle CloudInit setup
	if req.CloudInit {
		// Since this is a new VM creation, always set the CloudInit drive
		payload["ide2"] = "local:cloudinit"

		// Always set DHCP for CloudInit
		payload["ipconfig0"] = "ip=dhcp"

		if req.SSHKeys != "" {
			// Convert spaces to newlines for proper Proxmox format
			payload["sshkeys"] = strings.ReplaceAll(req.SSHKeys, " ", "\n")
		}

		if req.Nameserver != "" {
			payload["nameserver"] = req.Nameserver
		}

		if req.Searchdomain != "" {
			payload["searchdomain"] = req.Searchdomain
		}

		// Set CloudInit user if provided
		if req.Ciuser != "" {
			payload["ciuser"] = req.Ciuser
		}

		// Set CloudInit password if provided
		if req.Cipassword != "" {
			payload["cipassword"] = req.Cipassword
		}
	} else {
		// Use ISO if not using cloud-init
		payload["ide2"] = fmt.Sprintf("%s,media=cdrom", req.ISO)
	}

	return payload
}

func GetVMConfig(api *manager.APIManager, node, vmid string) (map[string]interface{}, error) {
	if _, err := strconv.Atoi(vmid); err != nil {
		return nil, fmt.Errorf("invalid VMID format")
	}

	response, err := api.ApiCall("GET", fmt.Sprintf("/nodes/%s/qemu/%s/config", node, vmid), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM config: %w", err)
	}

	return parseResponse(response)
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
