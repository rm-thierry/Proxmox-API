package main

import (
	"encoding/json"
	"fmt"
	"log"
	"rm-thierry/Proxmox-API/src/handlers"
	"rm-thierry/Proxmox-API/src/manager"
)

func main() {
	apiManager := manager.NewAPIManager()
	// router := gin.Default()
	// api.SetupRoutes(router, apiManager)
	// router.Run(":8080")
	// print("Server running on port 8080")

	// DBUser := os.Getenv("DB_USER")
	// DBPass := os.Getenv("DB_PASS")
	// DBName := os.Getenv("DB_NAME")
	// dbmanager, err := manager.NewDBManager(DBUser, DBPass, DBName)
	// if err != nil {
	// 	log.Fatalf("Error creating DB manager: %v", err)
	// }

	id, err := handlers.GetHighestVMID(apiManager, apiManager.Node)
	if err != nil {
		log.Fatalf("Error getting highest VM ID: %v", err)
	} else {
		fmt.Printf("Highest VM ID: %d\n", id)
	}

	config := handlers.NewDefaultVMConfig()
	config.VMID = fmt.Sprintf("%d", id)
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
