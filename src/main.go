package main

import (
	"rm-thierry/Proxmox-API/src/api"
	"rm-thierry/Proxmox-API/src/manager"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	apiManager := manager.NewAPIManager()
	api.SetupRoutes(router, apiManager)
	router.Run(":8080")
}
