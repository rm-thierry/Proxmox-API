package main

import (
	"encoding/json"
	"fmt"
	"log"
	"rm-thierry/Proxmox-API/src/handlers"
	"rm-thierry/Proxmox-API/src/manager"
)

func main() {
	// router := gin.Default()
	// api.SetupRoutes(router, apiManager)
	// router.Run(":8080")
	// print("Server running on port 8080")

	apiManager := manager.NewAPIManager()

	config := handlers.NewDefaultVMConfig()
	config.Name = "test-vm"

	result, err := handlers.CreateVM(apiManager, config)
	if err != nil {
		log.Fatalf("Error creating VM: %v", err)
	}

	prettyJSON, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		log.Fatalf("Error formatting JSON: %v", err)
	}

	fmt.Printf("\nVM Creation Result:\n%s\n", string(prettyJSON))
}
