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

	node, err := handlers.GetNode(apiManager, "pve")
	if err != nil {
		log.Fatalf("Error getting node: %v", err)
	}

	JSONNODES, err := json.MarshalIndent(node, "", "    ")
	if err != nil {
		log.Fatalf("Error marshaling node data: %v", err)
	}

	fmt.Println(string(JSONNODES))
}
