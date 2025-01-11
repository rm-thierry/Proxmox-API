package main

import (
	"encoding/json"
	"rm-thierry/Proxmox-API/src/handlers"
	"rm-thierry/Proxmox-API/src/manager"
)

func main() {
	apiManager := manager.NewAPIManager()
	vm, err := handlers.GetVM(apiManager, "pve", "103")
	if err != nil {
		panic(err)
	}

	JSONVMS, _ := json.MarshalIndent(vm, "", "  ")

	print(string(JSONVMS))

}
