package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

func main() {
	ports := []int{8001, 8002, 8003}
	var wg sync.WaitGroup

	for _, port := range ports {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			mux := http.NewServeMux()

			// Main handler
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				msg := fmt.Sprintf("👋 Hello from backend :%d!", p)
				fmt.Println(msg)
				w.Header().Set("Content-Type", "text/plain")
				fmt.Fprintln(w, msg)
			})

			// Health endpoint
			mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ok"))
			})

			// Metrics endpoint (returns GPU/CPU as JSON)
			mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]float64{
					"gpu_utilization": 25.0,
					"cpu_utilization": 30.0,
				})
			})

			addr := fmt.Sprintf(":%d", p)
			fmt.Printf("🚀 Backend server started on %s\n", addr)
			if err := http.ListenAndServe(addr, mux); err != nil {
				log.Fatalf("backend %s failed: %v", addr, err)
			}
		}(port)
	}

	wg.Wait()
}
