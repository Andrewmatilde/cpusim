# CPU仿真服务 - GCD计算器

一个使用Go编写的高性能GCD（最大公约数）计算服务，用于CPU负载仿真测试。

## 特性

- 使用固定的大整数（约300位）进行GCD计算
- 计算负载稳定，处理时间可预测
- 极简的HTTP API接口
- 轻量级二进制文件

## 项目结构

```text
cpusim/
├── bin/                # 编译输出目录
│   └── cpusim-server   # 可执行文件
├── cmd/
│   └── cpusim-server/  # 主程序入口
│       └── main.go
├── calculator/         # GCD计算模块
│   └── calculator.go
├── go.mod              # Go模块定义
├── Makefile            # 构建脚本
└── README.md           # 本文档
```

## 快速开始

### 编译

```bash
make build
```

编译后的二进制文件位于 `bin/cpusim-server`

### 运行

```bash
# 需要root权限绑定80端口
sudo ./bin/cpusim-server

# 或使用make
make run
```

### 清理

```bash
make clean
```

## API使用

### 计算接口

**端点:** `POST /calculate`

**请求示例:**

```bash
curl -X POST http://localhost:80/calculate \
  -H "Content-Type: application/json" \
  -d '{}'
```

**响应示例:**

```json
{
  "gcd": "1",
  "process_time": "2.356ms"
}
```

**响应字段说明:**

- `gcd`: 两个固定大整数的最大公约数
- `process_time`: 计算耗时

## 技术细节

### 固定整数

服务使用两个预定义的大整数（斐波那契数）：

- A: 300位整数
- B: 300位整数

每次请求都计算这两个固定整数的GCD，确保计算负载恒定。

### 算法

使用欧几里得算法（辗转相除法）计算GCD，对大整数使用Go的`math/big`包进行处理。

## Makefile命令

| 命令 | 说明 |
|------|------|
| `make build` | 编译服务到bin目录 |
| `make run` | 编译并运行服务（自动sudo） |
| `make clean` | 清理编译产物 |
| `make help` | 显示帮助信息 |

## 系统要求

- Go 1.22+
- Linux/macOS/Windows
- 需要root/sudo权限（绑定80端口）

## 性能特征

- 单次计算时间: ~2-5ms（取决于CPU性能）
- 内存占用: < 10MB
- 并发支持: 可处理大量并发请求

## 使用场景

- CPU性能基准测试
- 负载均衡测试
- 容器编排压力测试
- 微服务性能评估

## 许可

MIT License
