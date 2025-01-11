package api

import (
	"net/http"
	"rm-thierry/Proxmox-API/src/handlers"
	"rm-thierry/Proxmox-API/src/manager"

	"github.com/gin-gonic/gin"
)

type VMHandler struct {
	apiManager *manager.APIManager
}

func NewVMHandler(apiManager *manager.APIManager) *VMHandler {
	return &VMHandler{
		apiManager: apiManager,
	}
}

func SetupRoutes(router *gin.Engine, apiManager *manager.APIManager) {
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

func (h *VMHandler) CreateVM(c *gin.Context) {
	var config handlers.VMConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    result,
	})
}

type ISO struct {
	Debian string
	Ubuntu string
}

func GetISOs() ISO {
	return ISO{
		Debian: "local:iso/debian-12.8.0-amd64-netinst.iso",
		Ubuntu: "local:iso/ubuntu-22.04.3-live-server-amd64.iso",
	}
}

func (h *VMHandler) ListVMs(c *gin.Context) {
	node := c.DefaultQuery("node", "pve")

	vms, err := handlers.GetVMS(h.apiManager, node)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    vms,
	})
}

func (h *VMHandler) GetVM(c *gin.Context) {
	node := c.DefaultQuery("node", "pve")
	vmid := c.Param("vmid")

	vm, err := handlers.GetVM(h.apiManager, node, vmid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    vm,
	})
}

func (h *VMHandler) DeleteVM(c *gin.Context) {
	node := c.DefaultQuery("node", "pve")
	vmid := c.Param("vmid")

	result, err := handlers.DeleteVM(h.apiManager, node, vmid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func (h *VMHandler) StartVM(c *gin.Context) {
	node := c.DefaultQuery("node", "pve")
	vmid := c.Param("vmid")

	result, err := handlers.StartVM(h.apiManager, node, vmid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func (h *VMHandler) StopVM(c *gin.Context) {
	node := c.DefaultQuery("node", "pve")
	vmid := c.Param("vmid")

	result, err := handlers.StopVM(h.apiManager, node, vmid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}
