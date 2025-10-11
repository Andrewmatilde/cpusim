package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cpusim/dashboard/api/generated"
	"cpusim/pkg/dashboard"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

const (
	defaultPort        = "9090"
	defaultStoragePath = "./data/dashboard"
	defaultConfigPath  = "./configs/config.json"
)

func main() {
	// Get configuration from environment variables
	port := getEnv("PORT", defaultPort)
	configPath := getEnv("CONFIG_PATH", defaultConfigPath)
	storagePath := getEnv("STORAGE_PATH", defaultStoragePath)

	// Setup logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Load dashboard configuration from file
	config, err := loadDashboardConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load dashboard config: %v", err)
	}

	// Initialize dashboard service
	service, err := dashboard.NewService(storagePath, *config, logger)
	if err != nil {
		log.Fatalf("Failed to create dashboard service: %v", err)
	}

	// Initialize HTTP clients for sub-experiments
	err = initializeClients(service, config)
	if err != nil {
		log.Fatalf("Failed to initialize sub-experiment clients: %v", err)
	}

	// Create API handler
	apiHandler := &APIHandler{
		service: service,
		config:  *config,
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
		log.Printf("Starting dashboard server on port %s", port)
		log.Printf("Configuration loaded from: %s", configPath)
		log.Printf("Storage path: %s", storagePath)
		log.Printf("Target hosts: %d", len(config.TargetHosts))
		log.Printf("Client host: %s", config.ClientHost.Name)

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

// loadDashboardConfig loads dashboard configuration from JSON file
func loadDashboardConfig(path string) (*dashboard.Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config dashboard.Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return &config, nil
}

// initializeClients initializes HTTP clients for all sub-experiments
func initializeClients(service *dashboard.Service, config *dashboard.Config) error {
	// Initialize collector clients for each target host
	for _, target := range config.TargetHosts {
		client, err := dashboard.NewHTTPCollectorClient(target.CollectorServiceURL)
		if err != nil {
			return fmt.Errorf("failed to create collector client for %s: %w", target.Name, err)
		}
		service.SetCollectorClient(target.Name, client)
		log.Printf("Initialized collector client for %s (%s)", target.Name, target.CollectorServiceURL)
	}

	// Initialize requester client
	requesterClient, err := dashboard.NewHTTPRequesterClient(config.ClientHost.RequesterServiceURL)
	if err != nil {
		return fmt.Errorf("failed to create requester client: %w", err)
	}
	service.SetRequesterClient(requesterClient)
	log.Printf("Initialized requester client (%s)", config.ClientHost.RequesterServiceURL)

	return nil
}
