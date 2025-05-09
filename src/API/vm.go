package api

import (
	"net/http"
	"rm-thierry/Proxmox-API/src/auth"
	"rm-thierry/Proxmox-API/src/handlers"
	"rm-thierry/Proxmox-API/src/manager"
	"strconv"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type VMHandler struct {
	apiManager *manager.APIManager
}

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

type VMCloneRequest struct {
	SourceNode string `json:"source_node"`
	SourceVMID string `json:"source_vmid"`
	TargetNode string `json:"target_node"`
	TargetVMID string `json:"target_vmid"`
	Name       string `json:"name"`
}

func NewVMHandler(apiManager *manager.APIManager) *VMHandler {
	return &VMHandler{apiManager: apiManager}
}

func SetupRoutes(router *gin.Engine, apiManager *manager.APIManager, authService *auth.Service) {
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	router.Use(cors.New(corsConfig))

	handler := NewVMHandler(apiManager)

	// Apply authentication middleware to all API routes
	api := router.Group("/api/v1")
	api.Use(authService.AuthMiddleware())
	{
		// VM operations
		api.GET("/vms", handler.ListVMs)
		api.POST("/vms", handler.CreateVM)
		api.POST("/vms/template", handler.CreateVMFromTemplate)
		api.POST("/vms/clone", handler.CloneVM)
		api.GET("/vms/:vmid", handler.GetVM)
		api.DELETE("/vms/:vmid", handler.DeleteVM)
		api.POST("/vms/:vmid/start", handler.StartVM)
		api.POST("/vms/:vmid/stop", handler.StopVM)
		api.POST("/vms/:vmid/reboot", handler.RebootVM)

		// Resources and infrastructure
		api.GET("/resources", handler.GetResources)
		api.GET("/nodes", handler.GetNodes)
		api.GET("/storages", handler.GetStorages)
		api.GET("/networks", handler.GetNetworks)
		api.GET("/isos", handler.GetISOs)
		api.GET("/templates", handler.GetTemplates)
	}
}

func sendResponse(c *gin.Context, statusCode int, success bool, data interface{}, err string) {
	c.JSON(statusCode, Response{
		Success: success,
		Data:    data,
		Error:   err,
	})
}

func (h *VMHandler) ListVMs(c *gin.Context) {
	node := c.DefaultQuery("node", h.apiManager.Node)
	vms, err := handlers.ListVMs(h.apiManager, node)
	if err != nil {
		sendResponse(c, http.StatusInternalServerError, false, nil, "Failed to list VMs: "+err.Error())
		return
	}
	sendResponse(c, http.StatusOK, true, vms, "")
}

func (h *VMHandler) CreateVM(c *gin.Context) {
	var req VMCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendResponse(c, http.StatusBadRequest, false, nil, "Invalid request: "+err.Error())
		return
	}

	if req.Node == "" {
		req.Node = h.apiManager.Node
	}

	vm, err := handlers.CreateVM(h.apiManager, &handlers.VMCreateRequest{
		Node:         req.Node,
		VMID:         req.VMID,
		Name:         req.Name,
		Cores:        req.Cores,
		Memory:       req.Memory,
		Disk:         req.Disk,
		Net:          req.Net,
		ISO:          req.ISO,
		OSType:       req.OSType,
		CPU:          req.CPU,
		Sockets:      req.Sockets,
		Template:     req.Template,
		CloudInit:    req.CloudInit,
		SSHKeys:      req.SSHKeys,
		Nameserver:   req.Nameserver,
		Searchdomain: req.Searchdomain,
		Ciuser:       req.Ciuser,
		Cipassword:   req.Cipassword,
	})
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
		sendResponse(c, statusCode, false, nil, "Failed to create VM: "+err.Error())
		return
	}

	sendResponse(c, http.StatusCreated, true, vm, "")
}

func (h *VMHandler) CreateVMFromTemplate(c *gin.Context) {
	var req VMCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendResponse(c, http.StatusBadRequest, false, nil, "Invalid request: "+err.Error())
		return
	}

	if req.Node == "" {
		req.Node = h.apiManager.Node
	}

	if req.Template == "" {
		sendResponse(c, http.StatusBadRequest, false, nil, "Template name is required")
		return
	}

	// Set CloudInit to true by default when creating from template
	// This ensures the CloudInit settings will be applied
	req.CloudInit = true

	vm, err := handlers.CreateVMFromTemplate(h.apiManager, &handlers.VMCreateRequest{
		Node:         req.Node,
		VMID:         req.VMID,
		Name:         req.Name,
		Cores:        req.Cores,
		Memory:       req.Memory,
		Disk:         req.Disk,
		Net:          req.Net,
		Template:     req.Template,
		CloudInit:    req.CloudInit,
		SSHKeys:      req.SSHKeys,
		Nameserver:   req.Nameserver,
		Searchdomain: req.Searchdomain,
		Ciuser:       req.Ciuser,
		Cipassword:   req.Cipassword,
	})
	if err != nil {
		statusCode := http.StatusInternalServerError
		switch {
		case strings.Contains(err.Error(), "already exists"):
			statusCode = http.StatusConflict
		case strings.Contains(err.Error(), "invalid"):
			statusCode = http.StatusBadRequest
		case strings.Contains(err.Error(), "not found"):
			statusCode = http.StatusNotFound
		case strings.Contains(err.Error(), "template"):
			statusCode = http.StatusBadRequest
		}
		sendResponse(c, statusCode, false, nil, "Failed to create VM from template: "+err.Error())
		return
	}

	sendResponse(c, http.StatusCreated, true, vm, "")
}

func (h *VMHandler) GetVM(c *gin.Context) {
	node := c.DefaultQuery("node", h.apiManager.Node)
	vmid := c.Param("vmid")

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
		sendResponse(c, statusCode, false, nil, "Failed to get VM: "+err.Error())
		return
	}

	sendResponse(c, http.StatusOK, true, vm, "")
}

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
		sendResponse(c, statusCode, false, nil, "Failed to delete VM: "+err.Error())
		return
	}

	sendResponse(c, http.StatusOK, true, nil, "VM deleted successfully")
}

func (h *VMHandler) StartVM(c *gin.Context) {
	h.handleVMOperation(c, "start")
}

func (h *VMHandler) StopVM(c *gin.Context) {
	h.handleVMOperation(c, "stop")
}

func (h *VMHandler) RebootVM(c *gin.Context) {
	h.handleVMOperation(c, "reboot")
}

func (h *VMHandler) handleVMOperation(c *gin.Context, operation string) {
	// Get node and VMID from request
	node := c.DefaultQuery("node", h.apiManager.Node)
	vmid := c.Param("vmid")

	// Validate VMID
	if _, err := strconv.Atoi(vmid); err != nil {
		sendResponse(c, http.StatusBadRequest, false, nil, "VMID must be a number")
		return
	}

	// Perform the operation
	var err error
	switch operation {
	case "start":
		err = handlers.StartVM(h.apiManager, node, vmid)
	case "stop":
		err = handlers.StopVM(h.apiManager, node, vmid)
	case "reboot":
		err = handlers.RebootVM(h.apiManager, node, vmid)
	}

	// Handle errors
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}
		sendResponse(c, statusCode, false, nil, "Failed to "+operation+" VM: "+err.Error())
		return
	}

	// Return success response
	sendResponse(c, http.StatusOK, true, nil, "VM "+operation+" operation successful")
}

func (h *VMHandler) GetResources(c *gin.Context) {
	node := c.DefaultQuery("node", h.apiManager.Node)
	resources, err := handlers.GetResources(h.apiManager, node)
	if err != nil {
		sendResponse(c, http.StatusInternalServerError, false, nil,
			"Failed to get resources: "+err.Error())
		return
	}
	sendResponse(c, http.StatusOK, true, resources, "")
}

func (h *VMHandler) GetNodes(c *gin.Context) {
	nodes, err := handlers.GetNodes(h.apiManager)
	if err != nil {
		sendResponse(c, http.StatusInternalServerError, false, nil,
			"Failed to get nodes: "+err.Error())
		return
	}
	sendResponse(c, http.StatusOK, true, nodes, "")
}

func (h *VMHandler) GetStorages(c *gin.Context) {
	node := c.DefaultQuery("node", h.apiManager.Node)
	storages, err := handlers.GetStorages(h.apiManager, node)
	if err != nil {
		sendResponse(c, http.StatusInternalServerError, false, nil,
			"Failed to get storages: "+err.Error())
		return
	}
	sendResponse(c, http.StatusOK, true, storages, "")
}

func (h *VMHandler) GetNetworks(c *gin.Context) {
	node := c.DefaultQuery("node", h.apiManager.Node)
	networks, err := handlers.GetNetworks(h.apiManager, node)
	if err != nil {
		sendResponse(c, http.StatusInternalServerError, false, nil,
			"Failed to get networks: "+err.Error())
		return
	}
	sendResponse(c, http.StatusOK, true, networks, "")
}

func (h *VMHandler) CloneVM(c *gin.Context) {
	var req VMCloneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendResponse(c, http.StatusBadRequest, false, nil, "Invalid request: "+err.Error())
		return
	}

	// Set default values if not provided
	if req.SourceNode == "" {
		req.SourceNode = h.apiManager.Node
	}
	if req.TargetNode == "" {
		req.TargetNode = h.apiManager.Node
	}

	// Clone the VM
	result, err := handlers.CloneVM(h.apiManager, req.SourceNode, req.SourceVMID, req.TargetNode, req.TargetVMID, req.Name)
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
		sendResponse(c, statusCode, false, nil, "Failed to clone VM: "+err.Error())
		return
	}

	sendResponse(c, http.StatusCreated, true, result, "")
}

func (h *VMHandler) GetTemplates(c *gin.Context) {
	templates, err := handlers.GetVMTemplates()
	if err != nil {
		sendResponse(c, http.StatusInternalServerError, false, nil,
			"Failed to get templates: "+err.Error())
		return
	}
	sendResponse(c, http.StatusOK, true, templates, "")
}

func (h *VMHandler) GetISOs(c *gin.Context) {
	node := c.DefaultQuery("node", h.apiManager.Node)
	isos, err := handlers.GetISOs(h.apiManager, node)
	if err != nil {
		sendResponse(c, http.StatusInternalServerError, false, nil,
			"Failed to get ISOs: "+err.Error())
		return
	}
	sendResponse(c, http.StatusOK, true, isos, "")
}
