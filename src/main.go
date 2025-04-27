package main

import (
	"log"
	"os"
	api "rm-thierry/Proxmox-API/src/API"
	"rm-thierry/Proxmox-API/src/manager"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	// Database - attempt to load environment variables
	_ = godotenv.Load("env/.env")

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

	// Initialize API Manager
	apiManager := manager.NewAPIManager()

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
