# 项目设置
BINARY_NAME := cpusim-server
BINARY_PATH := bin/$(BINARY_NAME)
CMD_PATH := cmd/cpusim-server

# Collector 设置
COLLECTOR_BINARY_NAME := collector-server
COLLECTOR_BINARY_PATH := bin/$(COLLECTOR_BINARY_NAME)
COLLECTOR_CMD_PATH := cmd/collector-server

# Dashboard 设置
DASHBOARD_BINARY_NAME := dashboard-server
DASHBOARD_BINARY_PATH := bin/$(DASHBOARD_BINARY_NAME)
DASHBOARD_CMD_PATH := cmd/dashboard-server

# Go 编译器设置
GO := go
GOFLAGS := -v
LDFLAGS := -s -w

# 获取当前系统信息
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# 颜色输出
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m # No Color

.PHONY: all build run clean help build-collector run-collector run-both test-collector build-dashboard run-dashboard

# 默认目标
all: build

help:
	@echo "$(GREEN)可用的 Makefile 命令:$(NC)"
	@echo "  $(YELLOW)make build$(NC)           - 编译cpusim-server到 bin 目录"
	@echo "  $(YELLOW)make run$(NC)             - 编译并运行cpusim-server"
	@echo "  $(YELLOW)make build-collector$(NC) - 编译collector-server到 bin 目录"
	@echo "  $(YELLOW)make run-collector$(NC)   - 编译并运行collector-server"
	@echo "  $(YELLOW)make build-dashboard$(NC) - 编译dashboard-server到 bin 目录"
	@echo "  $(YELLOW)make run-dashboard$(NC)   - 编译并运行dashboard-server"
	@echo "  $(YELLOW)make run-both$(NC)        - 同时运行cpusim-server和collector-server"
	@echo "  $(YELLOW)make test-collector$(NC)  - 测试collector API"
	@echo "  $(YELLOW)make clean$(NC)           - 清理编译产物"
	@echo "  $(YELLOW)make help$(NC)            - 显示此帮助信息"

# 编译项目
build:
	@echo "$(GREEN)开始编译 $(BINARY_NAME)...$(NC)"
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_PATH) ./$(CMD_PATH)
	@echo "$(GREEN)编译完成！二进制文件位于: $(BINARY_PATH)$(NC)"
	@echo "系统架构: $(GOOS)/$(GOARCH)"

# 运行服务
run: build
	@echo "$(GREEN)启动服务（需要sudo权限绑定80端口）...$(NC)"
	sudo ./$(BINARY_PATH)

# 编译collector
build-collector:
	@echo "$(GREEN)开始编译 $(COLLECTOR_BINARY_NAME)...$(NC)"
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(COLLECTOR_BINARY_PATH) ./$(COLLECTOR_CMD_PATH)
	@echo "$(GREEN)编译完成！二进制文件位于: $(COLLECTOR_BINARY_PATH)$(NC)"

# 编译dashboard
build-dashboard:
	@echo "$(GREEN)开始编译 $(DASHBOARD_BINARY_NAME)...$(NC)"
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(DASHBOARD_BINARY_PATH) ./$(DASHBOARD_CMD_PATH)
	@echo "$(GREEN)编译完成！二进制文件位于: $(DASHBOARD_BINARY_PATH)$(NC)"

# 运行collector
run-collector: build-collector
	@echo "$(GREEN)启动collector服务（端口8080）...$(NC)"
	@mkdir -p data/experiments
	STORAGE_PATH=./data/experiments PORT=8080 CALCULATOR_PROCESS_NAME=cpusim-server ./$(COLLECTOR_BINARY_PATH)

# 运行dashboard
run-dashboard: build-dashboard
	@echo "$(GREEN)启动dashboard服务（端口9090）...$(NC)"
	@mkdir -p configs
	PORT=9090 CONFIG_PATH=./configs/config.json ./$(DASHBOARD_BINARY_PATH)

# 同时运行两个服务
run-both: build build-collector
	@echo "$(GREEN)同时启动两个服务...$(NC)"
	@mkdir -p data/experiments
	@echo "$(YELLOW)启动cpusim-server（端口80）...$(NC)"
	sudo ./$(BINARY_PATH) & \
	sleep 2 && \
	echo "$(YELLOW)启动collector-server（端口8080）...$(NC)" && \
	STORAGE_PATH=./data/experiments PORT=8080 CALCULATOR_PROCESS_NAME=cpusim-server ./$(COLLECTOR_BINARY_PATH)

# 测试collector API
test-collector:
	@echo "$(GREEN)测试collector API...$(NC)"
	@echo "$(YELLOW)1. 健康检查:$(NC)"
	curl -s http://localhost:8080/health | jq .
	@echo "\n$(YELLOW)2. 列出实验:$(NC)"
	curl -s http://localhost:8080/experiments | jq .
	@echo "\n$(YELLOW)3. 启动测试实验:$(NC)"
	curl -s -X POST http://localhost:8080/experiments \
		-H "Content-Type: application/json" \
		-d '{"experimentId": "$(shell uuidgen)", "description": "Test experiment", "timeout": 60, "collectionInterval": 1000}' | jq .

# 清理编译产物
clean:
	@echo "$(YELLOW)清理编译产物...$(NC)"
	@rm -rf bin/
	@rm -rf data/
	$(GO) clean
	@echo "$(GREEN)清理完成！$(NC)"