package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	api "rm-thierry/Proxmox-API/src/API"
	"rm-thierry/Proxmox-API/src/handlers"
	"rm-thierry/Proxmox-API/src/manager"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Set up command-line flags
	inputFile := flag.String("input", "", "Path to JSON input file")
	flag.Parse()

	// Load environment variables
	_ = godotenv.Load("env/.env")

	// Initialize API manager
	apiManager := manager.NewAPIManager()
	if apiManager.TokenID == "" || apiManager.TokenSecret == "" {
		log.Fatal("Error: Proxmox API credentials not found. Please set PROXMOX_TOKEN_ID and PROXMOX_TOKEN_SECRET environment variables.")
	}

	// Check if we're using file input
	if *inputFile != "" {
		processFileInput(*inputFile, apiManager)
		return
	}

	// Continue with API server setup if no input file provided
	// Database configuration - only initialize if all required environment variables are present
	dbHost := os.Getenv("DBHOST")
	dbUser := os.Getenv("DBUSER")
	dbPass := os.Getenv("DBPASS")
	dbName := os.Getenv("DBNAME")

	var dbManager *manager.DBManager

	if dbHost != "" && dbUser != "" && dbName != "" {
		config := manager.DBConfig{
			Host:     dbHost,
			Port:     3306,
			User:     dbUser,
			Password: dbPass,
			DBName:   dbName,
		}

		var err error
		dbManager, err = manager.NewDBManager(config)
		if err != nil {
			log.Printf("Warning: unable to connect to database: %v", err)
		} else {
			log.Println("Successfully connected to database at", dbHost)
			defer dbManager.Close()
		}
	} else {
		log.Println("Database connection skipped - environment variables not configured")
	}

	// Setup HTTP server
	router := gin.Default()
	api.SetupRoutes(router, apiManager)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting server on port %s", port)
	router.Run(":" + port)
}

func processFileInput(filename string, apiManager *manager.APIManager) {
	// Read the JSON file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Error reading input file: %v", err)
	}

	// Parse the JSON into a VM create request
	var vmRequest handlers.VMCreateRequest
	if err := json.Unmarshal(data, &vmRequest); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	// Set default node if not specified
	if vmRequest.Node == "" {
		vmRequest.Node = apiManager.Node
	}

	// Create the VM
	fmt.Println("Creating VM with the following configuration:")
	fmt.Printf("  VMID: %s\n", vmRequest.VMID)
	fmt.Printf("  Name: %s\n", vmRequest.Name)
	fmt.Printf("  Node: %s\n", vmRequest.Node)
	fmt.Printf("  Cores: %d\n", vmRequest.Cores)
	fmt.Printf("  Memory: %d MB\n", vmRequest.Memory)
	fmt.Printf("  Disk: %s\n", vmRequest.Disk)
	fmt.Printf("  Network: %s\n", vmRequest.Net)
	fmt.Printf("  ISO: %s\n", vmRequest.ISO)
	fmt.Printf("  OS Type: %s\n", vmRequest.OSType)
	fmt.Printf("  CPU: %s\n", vmRequest.CPU)
	fmt.Printf("  Sockets: %d\n", vmRequest.Sockets)

	vm, err := handlers.CreateVM(apiManager, &vmRequest)
	if err != nil {
		log.Fatalf("Error creating VM: %v", err)
	}

	// Print the result
	fmt.Println("VM created successfully!")
	resultJSON, _ := json.MarshalIndent(vm, "", "  ")
	fmt.Println(string(resultJSON))
}