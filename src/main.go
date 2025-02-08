package main

import (
	"log"
	"os"
	"rm-thierry/Proxmox-API/src/manager"

	"github.com/joho/godotenv"
)

func main() {

	//Databse

	err := godotenv.Load("env/.env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	config := manager.DBConfig{
		Host:     os.Getenv("DBHOST"),
		Port:     3306,
		User:     os.Getenv("DBUSER"),
		Password: os.Getenv("DBPASS"),
		DBName:   os.Getenv("DBNAME"),
	}

	dbManager, err := manager.NewDBManager(config)
	if err != nil {
		log.Fatalf("Fehler beim Verbindungsaufbau: %v", err)
	} else {
		log.Println("Verbindung zu", os.Getenv("DBHOST"), "erfolgreich!")
	}
	defer dbManager.Close()

	//API Manager

	//apiManager := manager.NewAPIManager()

	// id, err := handlers.GetHighestVMID(apiManager, apiManager.Node)
	// if err != nil {
	// 	log.Fatalf("Error getting highest VM ID: %v", err)
	// } else {
	// 	fmt.Printf("Highest VM ID: %d\n", id)
	// }

	// // config := handlers.NewDefaultVMConfig()
	// // config.VMID = fmt.Sprintf("%d", id)
	// // config.Name = "test-vm"
	// // config.Memory = "4096"
	// // config.Cores = "2"

	// // result, err := handlers.CreateVM(apiManager, config)
	// // if err != nil {
	// // 	log.Fatalf("Error creating VM: %v", err)
	// // }

	// // prettyJSON, _ := json.MarshalIndent(result, "", "    ")
	// // fmt.Printf("\nVM Creation Result:\n%s\n", string(prettyJSON))

	// containerConfig := handlers.NewDefaultContainerConfig()
	// containerConfig.CTID = "102"
	// containerConfig.Name = "test-container"
	// containerConfig.Disk = "100"
	// containerConfig.Memory = "2048"
	// containerConfig.Cores = "2"

	// result, err := handlers.CreateContainer(apiManager, containerConfig)
	// if err != nil {
	// 	log.Fatalf("Error creating container: %v", err)
	// }

	// prettyJSON, _ := json.MarshalIndent(result, "", "    ")
	// fmt.Printf("\nContainer Creation Result:\n%s\n", string(prettyJSON))
}
