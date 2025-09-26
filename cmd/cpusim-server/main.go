package main

import (
	"encoding/json"
	"log"
	"net/http"
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

func main() {
	// 初始化calculator并显示使用的固定数字信息
	calc := calculator.New()
	a, b := calc.GetFixedNumbers()
	log.Printf("GCD计算服务启动中，使用固定大整数:")
	log.Printf("A长度: %d位", len(a.String()))
	log.Printf("B长度: %d位", len(b.String()))

	http.HandleFunc("/calculate", calculateHandler)

	addr := ":80"
	log.Printf("监听端口 %s", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
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
