package api

import (
	"fmt"
	"net/http"
	"rm-thierry/Proxmox-API/src/handlers"
	"rm-thierry/Proxmox-API/src/manager"
	"strconv"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Response represents a standardized API response format
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// VMHandler handles VM-related API endpoints
type VMHandler struct {
	apiManager *manager.APIManager
}

// NewVMHandler creates a new VMHandler instance
func NewVMHandler(apiManager *manager.APIManager) *VMHandler {
	return &VMHandler{apiManager: apiManager}
}

// SetupRoutes configures all API routes
func SetupRoutes(router *gin.Engine, apiManager *manager.APIManager) {
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	router.Use(cors.New(corsConfig))

	handler := NewVMHandler(apiManager)

	api := router.Group("/api/v1")
	{
		// VM Management
		api.GET("/vms", handler.ListVMs)
		api.POST("/vms", handler.CreateVM)
		api.GET("/vms/:vmid", handler.GetVM)
		api.DELETE("/vms/:vmid", handler.DeleteVM)
		api.POST("/vms/:vmid/start", handler.StartVM)
		api.POST("/vms/:vmid/stop", handler.StopVM)
		api.POST("/vms/:vmid/reboot", handler.RebootVM)

		// Resource Information
		api.GET("/resources", handler.GetResources)
		api.GET("/nodes", handler.GetNodes)
		api.GET("/storages", handler.GetStorages)
		api.GET("/networks", handler.GetNetworks)
		api.GET("/isos", handler.GetISOs)
	}
}

// sendResponse sends a standardized JSON response
func sendResponse(c *gin.Context, statusCode int, success bool, data interface{}, err string) {
	c.JSON(statusCode, Response{
		Success: success,
		Data:    data,
		Error:   err,
	})
}

// ListVMs returns a list of all VMs on a node
func (h *VMHandler) ListVMs(c *gin.Context) {
	node := c.DefaultQuery("node", h.apiManager.Node)

	vms, err := handlers.ListVMs(h.apiManager, node)
	if err != nil {
		sendResponse(c, http.StatusInternalServerError, false, nil,
			fmt.Sprintf("Failed to list VMs: %v", err))
		return
	}

	sendResponse(c, http.StatusOK, true, vms, "")
}

// CreateVM creates a new virtual machine
func (h *VMHandler) CreateVM(c *gin.Context) {
	var req handlers.VMCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendResponse(c, http.StatusBadRequest, false, nil,
			fmt.Sprintf("Invalid request: %v", err))
		return
	}

	// Set default node if not specified
	if req.Node == "" {
		req.Node = h.apiManager.Node
	}

	// Validate and create VM
	vm, err := handlers.CreateVM(h.apiManager, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		switch {
		case strings.Contains(err.Error(), "already exists"):
			statusCode = http.StatusConflict
		case strings.Contains(err.Error(), "invalid"):
			statusCode = http.StatusBadRequest
		case strings.Contains(err.Error(), "not found"):
			statusCode = http.StatusNotFound
		}

		sendResponse(c, statusCode, false, nil, err.Error())
		return
	}

	sendResponse(c, http.StatusCreated, true, vm, "")
}

// GetVM returns details about a specific VM
func (h *VMHandler) GetVM(c *gin.Context) {
	node := c.DefaultQuery("node", h.apiManager.Node)
	vmid := c.Param("vmid")

	// Validate VMID
	if _, err := strconv.Atoi(vmid); err != nil {
		sendResponse(c, http.StatusBadRequest, false, nil, "VMID must be a number")
		return
	}

	vm, err := handlers.GetVM(h.apiManager, node, vmid)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}
		sendResponse(c, statusCode, false, nil, err.Error())
		return
	}

	sendResponse(c, http.StatusOK, true, vm, "")
}

// DeleteVM deletes a virtual machine
func (h *VMHandler) DeleteVM(c *gin.Context) {
	node := c.DefaultQuery("node", h.apiManager.Node)
	vmid := c.Param("vmid")

	if _, err := strconv.Atoi(vmid); err != nil {
		sendResponse(c, http.StatusBadRequest, false, nil, "VMID must be a number")
		return
	}

	if err := handlers.DeleteVM(h.apiManager, node, vmid); err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}
		sendResponse(c, statusCode, false, nil, err.Error())
		return
	}

	sendResponse(c, http.StatusOK, true, nil, "VM deleted successfully")
}

// StartVM starts a virtual machine
func (h *VMHandler) StartVM(c *gin.Context) {
	h.handleVMOperation(c, "start")
}

// StopVM stops a virtual machine
func (h *VMHandler) StopVM(c *gin.Context) {
	h.handleVMOperation(c, "stop")
}

// RebootVM reboots a virtual machine
func (h *VMHandler) RebootVM(c *gin.Context) {
	h.handleVMOperation(c, "reboot")
}

// handleVMOperation handles common VM operations (start/stop/reboot)
func (h *VMHandler) handleVMOperation(c *gin.Context, operation string) {
	node := c.DefaultQuery("node", h.apiManager.Node)
	vmid := c.Param("vmid")

	if _, err := strconv.Atoi(vmid); err != nil {
		sendResponse(c, http.StatusBadRequest, false, nil, "VMID must be a number")
		return
	}

	var err error
	switch operation {
	case "start":
		err = handlers.StartVM(h.apiManager, node, vmid)
	case "stop":
		err = handlers.StopVM(h.apiManager, node, vmid)
	case "reboot":
		err = handlers.RebootVM(h.apiManager, node, vmid)
	}

	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}
		sendResponse(c, statusCode, false, nil, err.Error())
		return
	}

	sendResponse(c, http.StatusOK, true, nil, fmt.Sprintf("VM %s operation successful", operation))
}

// GetResources returns available resources on a node
func (h *VMHandler) GetResources(c *gin.Context) {
	node := c.DefaultQuery("node", h.apiManager.Node)
	resources, err := handlers.GetResources(h.apiManager, node)
	if err != nil {
		sendResponse(c, http.StatusInternalServerError, false, nil,
			fmt.Sprintf("Failed to get resources: %v", err))
		return
	}
	sendResponse(c, http.StatusOK, true, resources, "")
}

// GetNodes returns a list of all nodes in the cluster
func (h *VMHandler) GetNodes(c *gin.Context) {
	nodes, err := handlers.GetNodes(h.apiManager)
	if err != nil {
		sendResponse(c, http.StatusInternalServerError, false, nil,
			fmt.Sprintf("Failed to get nodes: %v", err))
		return
	}
	sendResponse(c, http.StatusOK, true, nodes, "")
}

// GetStorages returns available storage on a node
func (h *VMHandler) GetStorages(c *gin.Context) {
	node := c.DefaultQuery("node", h.apiManager.Node)
	storages, err := handlers.GetStorages(h.apiManager, node)
	if err != nil {
		sendResponse(c, http.StatusInternalServerError, false, nil,
			fmt.Sprintf("Failed to get storages: %v", err))
		return
	}
	sendResponse(c, http.StatusOK, true, storages, "")
}

// GetNetworks returns available networks on a node
func (h *VMHandler) GetNetworks(c *gin.Context) {
	node := c.DefaultQuery("node", h.apiManager.Node)
	networks, err := handlers.GetNetworks(h.apiManager, node)
	if err != nil {
		sendResponse(c, http.StatusInternalServerError, false, nil,
			fmt.Sprintf("Failed to get networks: %v", err))
		return
	}
	sendResponse(c, http.StatusOK, true, networks, "")
}

// GetISOs returns available ISOs on a node
func (h *VMHandler) GetISOs(c *gin.Context) {
	node := c.DefaultQuery("node", h.apiManager.Node)
	isos, err := handlers.GetISOs(h.apiManager, node)
	if err != nil {
		sendResponse(c, http.StatusInternalServerError, false, nil,
			fmt.Sprintf("Failed to get ISOs: %v", err))
		return
	}
	sendResponse(c, http.StatusOK, true, isos, "")
}
