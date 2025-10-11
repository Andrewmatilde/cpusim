package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"cpusim/collector/api/generated"
	"cpusim/pkg/collector"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

const (
	defaultPort               = "8080"
	defaultCollectionInterval = "1"
	defaultCalculatorProcess  = "cpusim-server"
	defaultStoragePath        = "./data/collector"
)

func main() {
	// Get configuration from environment variables
	port := getEnv("PORT", defaultPort)

	// Setup logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Create collector config from environment
	collectionInterval, _ := strconv.Atoi(getEnv("COLLECTION_INTERVAL", defaultCollectionInterval))

	config := collector.Config{
		CollectionInterval: collectionInterval,
		CalculatorProcess:  getEnv("CALCULATOR_PROCESS", defaultCalculatorProcess),
	}

	storagePath := getEnv("STORAGE_PATH", defaultStoragePath)

	// Initialize collector service
	service, err := collector.NewService(storagePath, config, logger)
	if err != nil {
		log.Fatalf("Failed to create collector service: %v", err)
	}

	// Create API handler
	apiHandler := &APIHandler{
		service: service,
		config:  config,
		logger:  logger,
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
		log.Printf("Collection interval: %d seconds", config.CollectionInterval)
		log.Printf("Calculator process: %s", config.CalculatorProcess)
		log.Printf("Storage path: %s", storagePath)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Stop current running experiment
	if err := service.StopExperiment(); err != nil {
		log.Printf("Error stopping experiment: %v", err)
	}

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
