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
	config := handlers.NewDefaultVMConfig()
	config.VMID = "111"
	config.Name = "test-vm"
	config.Cores = "2"
	config.Memory = "4096"
	config.ISO = handlers.GetISOs().Debian

	result, err := handlers.CreateVM(apiManager, config)
	if err != nil {
		log.Fatalf("Error creating VM: %v", err)
	}

	JSONNODES, _ := json.MarshalIndent(result, "", " ")
	fmt.Println(string(JSONNODES))
}
