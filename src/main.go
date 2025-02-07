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

	id, err := handlers.GetHighestVMID(apiManager, apiManager.Node)
	if err != nil {
		log.Fatalf("Error getting highest VM ID: %v", err)
	} else {
		fmt.Printf("Highest VM ID: %d\n", id)
	}

	config := handlers.NewDefaultVMConfig()
	//	config.VMID = fmt.Sprintf("%d", id)
	config.VMID = "150"
	config.Name = "test-vm"
	config.Memory = "4096"
	config.Cores = "2"

	result, err := handlers.CreateVM(apiManager, config)
	if err != nil {
		log.Fatalf("Error creating VM: %v", err)
	}

	prettyJSON, _ := json.MarshalIndent(result, "", "    ")
	fmt.Printf("\nVM Creation Result:\n%s\n", string(prettyJSON))
}
