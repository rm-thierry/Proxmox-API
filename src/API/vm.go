package api

import (
	"net/http"
	"rm-thierry/Proxmox-API/src/handlers"
	"rm-thierry/Proxmox-API/src/manager"

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

func NewVMHandler(apiManager *manager.APIManager) *VMHandler {
	return &VMHandler{apiManager: apiManager}
}

func SetupRoutes(router *gin.Engine, apiManager *manager.APIManager) {
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"}
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
		defaultConfig := handlers.NewDefaultVMConfig()
		if config.VMID == "" {
			config.VMID = defaultConfig.VMID
		}
		if config.Name == "" {
			config.Name = defaultConfig.Name
		}
	}

	isos := handlers.GetISOs()
	switch config.OSType {
	case "debian":
		config.ISO = isos.Debian
	default:
		config.ISO = handlers.NewDefaultVMConfig().ISO
	}

	result, err := handlers.CreateVM(h.apiManager, config)
	if err != nil {
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
