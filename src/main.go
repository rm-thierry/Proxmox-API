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
	node := "pve"
	vmid := "100"

	vmDetails, err := handlers.GetVM(apiManager, node, vmid)
	if err != nil {
		log.Fatalf("Error fetching VM details: %v", err)
	}

	JSON, _ := json.MarshalIndent(vmDetails, "", "  ")

	fmt.Printf("VM Details: %s\n", JSON)
}
