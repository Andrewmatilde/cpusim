package main

import (
	"log"
	"os"

	"cpusim/dashboard/api/generated"
	"cpusim/dashboard/pkg/config"
	"cpusim/dashboard/pkg/handlers"
	"cpusim/dashboard/pkg/services"

	"github.com/gin-gonic/gin"
)

func main() {
	// 从环境变量获取配置
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./configs/config.json"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	// 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建服务和处理器
	dashboardService := services.NewDashboardService(cfg)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)

	// 创建Gin路由器
	r := gin.Default()

	// 注册OpenAPI生成的路由
	generated.RegisterHandlers(r, dashboardHandler)

	// 启动服务器
	log.Printf("Starting dashboard server on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}