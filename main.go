package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "HTTP request latency",
		},
		[]string{"method", "endpoint"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

func metricsMiddleware(next http.HandlerFunc, endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		next.ServeHTTP(w, r)
		
		duration := time.Since(start).Seconds()
		httpRequestsTotal.WithLabelValues(r.Method, endpoint, "200").Inc()
		httpRequestDuration.WithLabelValues(r.Method, endpoint).Observe(duration)
	}
}







type SampleRequest struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

type SampleResponse struct {
	ID      int    `json:"id"`
	Status  string `json:"status"`
	Echo    string `json:"echo"`
}

func samplePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SampleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response := SampleResponse{
		ID:     123,
		Status: "success",
		Echo:   fmt.Sprintf("Hello %s: %s", req.Name, req.Message),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Metrics server on dedicated port
	metricsServeMux := http.NewServeMux()
	metricsServeMux.Handle("/metrics", promhttp.Handler())
	
	go func() {
		log.Println("Metrics server starting on :9090")
		log.Fatal(http.ListenAndServe(":9090", metricsServeMux))
	}()

	// Main server
	mainServeMux := http.NewServeMux()
	mainServeMux.HandleFunc("/api/sample2", metricsMiddleware(samplePostHandler, "/api/sample2"))
	
	log.Println("Main server starting on :8080")
	log.Println("Endpoints available:")
	log.Println("  POST /api/sample2 - Sample endpoint")
	log.Println("Metrics server available on :9090/metrics")
	log.Fatal(http.ListenAndServe(":8080", mainServeMux))
}
