package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"cpusim/calculator"
)

type CalculationRequest struct {
	// 请求体可以为空，因为使用固定值
}

type CalculationResponse struct {
	GCD         string `json:"gcd"`
	ProcessTime string `json:"process_time"`
}

// BenchmarkStats stores benchmark statistics
type BenchmarkStats struct {
	totalCount   int64
	totalDuration int64 // in nanoseconds
}

func (s *BenchmarkStats) recordCalculation(duration time.Duration) {
	atomic.AddInt64(&s.totalCount, 1)
	atomic.AddInt64(&s.totalDuration, duration.Nanoseconds())
}

func (s *BenchmarkStats) snapshot() (count int64, avgDuration time.Duration) {
	count = atomic.LoadInt64(&s.totalCount)
	totalNs := atomic.LoadInt64(&s.totalDuration)
	if count > 0 {
		avgDuration = time.Duration(totalNs / count)
	}
	return
}

func main() {
	// Command line flags
	mode := flag.String("mode", "server", "运行模式: server (HTTP服务器) 或 benchmark (性能测试)")
	duration := flag.Int("duration", 10, "benchmark模式下的运行时长（秒）")
	concurrency := flag.Int("concurrency", 1, "benchmark模式下的并发数")
	port := flag.Int("port", 80, "server模式下的监听端口")
	flag.Parse()

	// 初始化calculator并显示使用的固定数字信息
	calc := calculator.New()
	a, b := calc.GetFixedNumbers()
	log.Printf("GCD计算服务启动中，使用固定大整数:")
	log.Printf("A长度: %d位", len(a.String()))
	log.Printf("B长度: %d位", len(b.String()))

	switch *mode {
	case "server":
		runServerMode(*port)
	case "benchmark":
		runBenchmarkMode(*duration, *concurrency)
	default:
		log.Fatalf("未知模式: %s (支持: server, benchmark)", *mode)
	}
}

// runServerMode runs the HTTP server mode
func runServerMode(port int) {
	http.HandleFunc("/calculate", calculateHandler)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Server模式: 监听端口 %s", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}

// runBenchmarkMode runs the benchmark mode
func runBenchmarkMode(durationSec, concurrency int) {
	log.Printf("Benchmark模式: 运行 %d 秒, 并发 %d", durationSec, concurrency)

	stats := &BenchmarkStats{}
	deadline := time.Now().Add(time.Duration(durationSec) * time.Second)
	doneChan := make(chan struct{})

	// Start worker goroutines
	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			calc := calculator.New()
			count := 0
			for time.Now().Before(deadline) {
				start := time.Now()
				_ = calc.GCD()
				duration := time.Since(start)
				stats.recordCalculation(duration)
				count++
			}
			log.Printf("Worker %d: 完成 %d 次计算", workerID, count)
		}(i)
	}

	// Progress reporter
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		lastCount := int64(0)
		for {
			select {
			case <-ticker.C:
				count, avgDuration := stats.snapshot()
				qps := count - lastCount
				lastCount = count
				log.Printf("进度: 总计=%d, QPS=%d, 平均延迟=%v", count, qps, avgDuration)
			case <-doneChan:
				return
			}
		}
	}()

	// Wait for deadline
	time.Sleep(time.Duration(durationSec) * time.Second)
	close(doneChan)
	time.Sleep(100 * time.Millisecond) // Give workers time to finish

	// Final statistics
	totalCount, avgDuration := stats.snapshot()
	totalQPS := float64(totalCount) / float64(durationSec)

	log.Println("\n=== Benchmark 结果 ===")
	log.Printf("总计算次数: %d", totalCount)
	log.Printf("运行时长: %d 秒", durationSec)
	log.Printf("平均 QPS: %.2f", totalQPS)
	log.Printf("平均延迟: %v", avgDuration)
	log.Printf("并发数: %d", concurrency)
	log.Println("====================")
}

func calculateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST请求", http.StatusMethodNotAllowed)
		return
	}

	startTime := time.Now()

	// 创建calculator实例并计算GCD
	calc := calculator.New()
	result := calc.GCD()

	processTime := time.Since(startTime)

	response := CalculationResponse{
		GCD:         result.String(),
		ProcessTime: processTime.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
