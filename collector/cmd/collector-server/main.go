package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"cpusim/collector/api/generated"
	"cpusim/collector/pkg/experiment"
	"cpusim/collector/pkg/metrics"
	"cpusim/collector/pkg/storage"

	"github.com/gin-gonic/gin"
)

const (
	defaultPort                = "8080"
	defaultCalculatorProcessName = "cpusim-server"
	defaultStoragePath         = "./data/experiments"
	defaultCollectionInterval  = 1000 // milliseconds
	defaultTimeout            = 300   // seconds
)

func main() {
	// Get configuration from environment variables
	port := getEnv("PORT", defaultPort)
	calculatorProcessName := getEnv("CALCULATOR_PROCESS_NAME", defaultCalculatorProcessName)
	storagePath := getEnv("STORAGE_PATH", defaultStoragePath)

	// Create storage directory
	absStoragePath, err := filepath.Abs(storagePath)
	if err != nil {
		log.Fatalf("Failed to get absolute storage path: %v", err)
	}

	// Initialize components
	storage, err := storage.NewFileStorage(absStoragePath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	metricsCollector := metrics.NewCollector(calculatorProcessName)
	experimentManager := experiment.NewManager(metricsCollector, storage)

	// Create API handler
	apiHandler := &APIHandler{
		experimentManager: experimentManager,
		storage:          storage,
	}

	// Set up Gin router
	if gin.Mode() == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Register OpenAPI generated routes
	generated.RegisterHandlers(router, apiHandler)

	// Create HTTP server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting collector server on port %s", port)
		log.Printf("Calculator process name: %s", calculatorProcessName)
		log.Printf("Storage path: %s", absStoragePath)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Give the server 30 seconds to finish the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}