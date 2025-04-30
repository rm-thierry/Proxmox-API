package api

import (
	"fmt"
	"log"
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

	if config.Node == "" {
		config.Node = h.apiManager.Node
		log.Printf("No node specified, using default: %s", config.Node)
	}

	if config.VMID == "" || config.Name == "" {
		if config.VMID == "" {
			highestID, err := handlers.GetHighestVMID(h.apiManager, config.Node)
			if err != nil {
				sendResponse(c, http.StatusInternalServerError, false, nil, fmt.Sprintf("Error getting highest VM ID: %v", err))
				return
			}
			config.VMID = fmt.Sprintf("%d", highestID)
			log.Printf("Generated new VMID: %s", config.VMID)
		}
		if config.Name == "" {
			config.Name = "vm-" + config.VMID
			log.Printf("Generated VM name: %s", config.Name)
		}
	}

	if config.Cores == "" {
		config.Cores = "2"
	}
	if config.Memory == "" {
		config.Memory = "4096"
	}
	if config.Disk == "" {
		config.Disk = "local-lvm:32"
	}
	if config.Net == "" {
		config.Net = "vmbr0"
	}
	if config.OSType == "" {
		config.OSType = "l26"
	} else if config.OSType == "debian" || config.OSType == "ubuntu" {
		config.OSType = "l26"
	} else if config.OSType == "windows" {
		config.OSType = "win10"
	}
	if config.CPU == "" {
		config.CPU = "host"
	}
	if config.Sockets == "" {
		config.Sockets = "1"
	}

	isos := handlers.GetISOs(h.apiManager, config.Node)
	if config.ISO == "" {
		if strings.HasPrefix(config.OSType, "w") {
			config.ISO = isos.Windows
		} else if config.OSType == "l26" || config.OSType == "l24" {
			config.ISO = isos.Debian
		} else {
			config.ISO = isos.Debian
		}
	}

	resources, _ := handlers.GetAvailableResources(h.apiManager, config.Node)

	resourceISOs := []string{}
	if isosVal, ok := resources["isos"].([]string); ok && len(isosVal) > 0 {
		resourceISOs = isosVal
	}

	checkISOs := resourceISOs
	if len(checkISOs) == 0 {
		checkISOs = isos.All
	}

	log.Printf("Checking ISO: %s", config.ISO)
	log.Printf("Available ISOs: %v", checkISOs)

	isoValidated := false
	if len(checkISOs) > 0 {
		for _, availableISO := range checkISOs {
			if config.ISO == availableISO {
				isoValidated = true
				break
			}
		}
	}

	if !isoValidated {
		if strings.HasPrefix(config.ISO, "local:iso/") {
			log.Printf("ISO %s not in available list but has expected format - bypassing validation", config.ISO)
			isoValidated = true
		} else {
			availableISOsStr := strings.Join(checkISOs, ", ")
			errorMsg := fmt.Sprintf("ISO %s not found. Available ISOs: [%s]. Use GET /api/v1/resources for complete resource information",
				config.ISO, availableISOsStr)
			sendResponse(c, http.StatusBadRequest, false, nil, errorMsg)
			return
		}
	}

	networkValidated := false
	if networksVal, ok := resources["networkNames"].([]string); ok {
		isCommonBridge := config.Net == "vmbr0" || config.Net == "vmbr1"

		if isCommonBridge {
			networkValidated = true
		} else {
			for _, network := range networksVal {
				if config.Net == network {
					networkValidated = true
					break
				}
			}

			if !networkValidated && len(networksVal) > 0 {
				availableNetworksStr := strings.Join(networksVal, ", ")
				errorMsg := fmt.Sprintf("Network bridge %s not found. Available bridges: [%s]. Use GET /api/v1/resources for complete resource information",
					config.Net, availableNetworksStr)
				sendResponse(c, http.StatusBadRequest, false, nil, errorMsg)
				return
			}
		}
	} else {
		if config.Net == "vmbr0" || config.Net == "vmbr1" {
			networkValidated = true
		}
	}

	log.Printf("VM configuration validated, attempting to create VM with ID: %s, Name: %s", config.VMID, config.Name)

	result, err := handlers.CreateVM(h.apiManager, config)
	if err != nil {
		log.Printf("Error in CreateVM: %v", err)

		if strings.Contains(err.Error(), "Available storages:") ||
			strings.Contains(err.Error(), "Available bridges:") ||
			strings.Contains(err.Error(), "Available ISOs:") {
			sendResponse(c, http.StatusBadRequest, false, nil, err.Error())
			return
		}

		if strings.Contains(err.Error(), "Status 501") {
			errorMsg := "The Proxmox API returned a 'Not Implemented' error (501). This usually means one of the following issues:\n" +
				"1. The VM parameters are not correctly formatted for this version of Proxmox\n" +
				"2. The storage format is not supported or missing required parameters\n" +
				"3. Required feature is not available on this Proxmox installation"

			sendResponse(c, http.StatusInternalServerError, false, nil, fmt.Sprintf("%s\nOriginal error: %v", errorMsg, err))
			return
		}

		sendResponse(c, http.StatusInternalServerError, false, nil, err.Error())
		return
	}

	log.Printf("VM created successfully: %v", result)
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