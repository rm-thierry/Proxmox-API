package api

import (
	"fmt"
	"net/http"
	"rm-thierry/Proxmox-API/src/handlers"
	"rm-thierry/Proxmox-API/src/manager"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Helper function to check if a string is in a slice
func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

type VMHandler struct {
	apiManager *manager.APIManager
}

func NewVMHandler(apiManager *manager.APIManager) *VMHandler {
	return &VMHandler{apiManager: apiManager}
}

func SetupRoutes(router *gin.Engine, apiManager *manager.APIManager) {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}

	router.Use(cors.New(config))

	handler := NewVMHandler(apiManager)
	v1 := router.Group("/api/v1")
	{
		vms := v1.Group("/vms")
		{
			vms.GET("", handler.ListVMs)
			vms.POST("", handler.CreateVM)
			vms.GET("/:vmid", handler.GetVM)
			vms.DELETE("/:vmid", handler.DeleteVM)
			vms.POST("/:vmid/start", handler.StartVM)
			vms.POST("/:vmid/stop", handler.StopVM)
		}
		
		// Resources endpoint
		v1.GET("/resources", handler.GetResources)
	}
}

func sendResponse(c *gin.Context, statusCode int, success bool, data interface{}, err string) {
	c.JSON(statusCode, Response{
		Success: success,
		Data:    data,
		Error:   err,
	})
}

func (h *VMHandler) CreateVM(c *gin.Context) {
	var config handlers.VMConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		sendResponse(c, http.StatusBadRequest, false, nil, "Invalid request body")
		return
	}

	if config.VMID == "" || config.Name == "" {
		if config.VMID == "" {
			highestID, err := handlers.GetHighestVMID(h.apiManager, h.apiManager.Node)
			if err != nil {
				sendResponse(c, http.StatusInternalServerError, false, nil, fmt.Sprintf("Error getting highest VM ID: %v", err))
				return
			}
			config.VMID = fmt.Sprintf("%d", highestID)
		}
		if config.Name == "" {
			config.Name = "vm-" + config.VMID
		}
	}

	isos := handlers.GetISOs(h.apiManager, h.apiManager.Node)
	switch config.OSType {
	case "debian":
		config.ISO = isos.Debian
	case "ubuntu":
		config.ISO = isos.Ubuntu
	case "windows":
		config.ISO = isos.Windows
	default:
		config.ISO = isos.Debian
	}
	
	// If requested ISO not available, include a list of available ISOs in error
	if !stringInSlice(config.ISO, isos.All) && len(isos.All) > 0 {
		availableISOsStr := strings.Join(isos.All, ", ")
		errorMsg := fmt.Sprintf("ISO %s not found. Available ISOs: [%s]. Use GET /api/v1/resources for complete resource information", config.ISO, availableISOsStr)
		sendResponse(c, http.StatusBadRequest, false, nil, errorMsg)
		return
	}

	// Check network bridge
	resources, _ := handlers.GetAvailableResources(h.apiManager, h.apiManager.Node)
	if networksVal, ok := resources["networkNames"].([]string); ok {
		// Only validate if network isn't the default "vmbr0" and we have network info
		if config.Net != "vmbr0" && len(networksVal) > 0 && !stringInSlice(config.Net, networksVal) {
			availableNetworksStr := strings.Join(networksVal, ", ")
			errorMsg := fmt.Sprintf("Network bridge %s not found. Available bridges: [%s]. Use GET /api/v1/resources for complete resource information", config.Net, availableNetworksStr)
			sendResponse(c, http.StatusBadRequest, false, nil, errorMsg)
			return
		}
	}

	result, err := handlers.CreateVM(h.apiManager, config)
	if err != nil {
		// Check if the error contains information about available resources
		if strings.Contains(err.Error(), "Available storages:") || 
		   strings.Contains(err.Error(), "Available bridges:") ||
		   strings.Contains(err.Error(), "Available ISOs:") {
			// Return a 400 Bad Request with the list of available resources
			sendResponse(c, http.StatusBadRequest, false, nil, err.Error())
			return
		}
		
		// For other errors, return 500 Internal Server Error
		sendResponse(c, http.StatusInternalServerError, false, nil, err.Error())
		return
	}

	sendResponse(c, http.StatusCreated, true, result, "")
}

func (h *VMHandler) ListVMs(c *gin.Context) {
	node := c.DefaultQuery("node", h.apiManager.Node)
	vms, err := handlers.GetVMS(h.apiManager, node)
	if err != nil {
		sendResponse(c, http.StatusInternalServerError, false, nil, err.Error())
		return
	}
	sendResponse(c, http.StatusOK, true, vms, "")
}

func (h *VMHandler) GetVM(c *gin.Context) {
	node := c.DefaultQuery("node", h.apiManager.Node)
	vmid := c.Param("vmid")
	vm, err := handlers.GetVM(h.apiManager, node, vmid)
	if err != nil {
		sendResponse(c, http.StatusInternalServerError, false, nil, err.Error())
		return
	}
	sendResponse(c, http.StatusOK, true, vm, "")
}

func (h *VMHandler) DeleteVM(c *gin.Context) {
	node := c.DefaultQuery("node", h.apiManager.Node)
	vmid := c.Param("vmid")
	result, err := handlers.DeleteVM(h.apiManager, node, vmid)
	if err != nil {
		sendResponse(c, http.StatusInternalServerError, false, nil, err.Error())
		return
	}
	sendResponse(c, http.StatusOK, true, result, "")
}

func (h *VMHandler) StartVM(c *gin.Context) {
	node := c.DefaultQuery("node", h.apiManager.Node)
	vmid := c.Param("vmid")
	result, err := handlers.StartVM(h.apiManager, node, vmid)
	if err != nil {
		sendResponse(c, http.StatusInternalServerError, false, nil, err.Error())
		return
	}
	sendResponse(c, http.StatusOK, true, result, "")
}

func (h *VMHandler) StopVM(c *gin.Context) {
	node := c.DefaultQuery("node", h.apiManager.Node)
	vmid := c.Param("vmid")
	result, err := handlers.StopVM(h.apiManager, node, vmid)
	if err != nil {
		sendResponse(c, http.StatusInternalServerError, false, nil, err.Error())
		return
	}
	sendResponse(c, http.StatusOK, true, result, "")
}

func (h *VMHandler) GetResources(c *gin.Context) {
	node := c.DefaultQuery("node", h.apiManager.Node)
	resources, err := handlers.GetAvailableResources(h.apiManager, node)
	if err != nil {
		sendResponse(c, http.StatusInternalServerError, false, nil, err.Error())
		return
	}
	sendResponse(c, http.StatusOK, true, resources, "")
}