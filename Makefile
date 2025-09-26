# 项目设置
BINARY_NAME := cpusim-server
BINARY_PATH := bin/$(BINARY_NAME)
CMD_PATH := cmd/cpusim-server

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

.PHONY: all build run clean help

# 默认目标
all: build

help:
	@echo "$(GREEN)可用的 Makefile 命令:$(NC)"
	@echo "  $(YELLOW)make build$(NC)  - 编译服务到 bin 目录"
	@echo "  $(YELLOW)make run$(NC)    - 编译并运行服务"
	@echo "  $(YELLOW)make clean$(NC)  - 清理编译产物"
	@echo "  $(YELLOW)make help$(NC)   - 显示此帮助信息"

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

# 清理编译产物
clean:
	@echo "$(YELLOW)清理编译产物...$(NC)"
	@rm -rf bin/
	$(GO) clean
	@echo "$(GREEN)清理完成！$(NC)"