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

	"cpusim/pkg/requester"
	"cpusim/requester/api/generated"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

const (
	defaultPort       = "80"
	defaultTargetIP   = "localhost"
	defaultTargetPort = "8080"
	defaultQPS        = "10"
	defaultTimeout    = "30"
	defaultStoragePath = "./data/requester"
)

func main() {
	// Get configuration from environment variables
	port := getEnv("PORT", defaultPort)

	// Setup logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Create requester config from environment
	targetPort, _ := strconv.Atoi(getEnv("TARGET_PORT", defaultTargetPort))
	qps, _ := strconv.Atoi(getEnv("QPS", defaultQPS))
	timeout, _ := strconv.Atoi(getEnv("TIMEOUT", defaultTimeout))

	config := requester.Config{
		TargetIP:   getEnv("TARGET_IP", defaultTargetIP),
		TargetPort: targetPort,
		QPS:        qps,
		Timeout:    timeout,
	}

	storagePath := getEnv("STORAGE_PATH", defaultStoragePath)

	// Initialize requester service
	service, err := requester.NewService(storagePath, config, logger)
	if err != nil {
		log.Fatalf("Failed to create requester service: %v", err)
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
		log.Printf("Starting requester server on port %s", port)

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