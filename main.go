package main

import (
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
		[]string{"endpoint", "method", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "Duration of HTTP requests in seconds",
		},
		[]string{"endpoint", "method"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

func prometheusMiddleware(next http.HandlerFunc, endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		next.ServeHTTP(w, r)
		
		duration := time.Since(start).Seconds()
		httpRequestDuration.WithLabelValues(endpoint, r.Method).Observe(duration)
		httpRequestsTotal.WithLabelValues(endpoint, r.Method, "200").Inc()
	}
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World! 🌍\n")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK\n")
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Go Hello World Server with Prometheus metrics\nVersion: 1.0.0\n")
}

func main() {
	http.HandleFunc("/", prometheusMiddleware(helloHandler, "hello"))
	http.HandleFunc("/health", prometheusMiddleware(healthHandler, "health"))
	http.HandleFunc("/about", prometheusMiddleware(aboutHandler, "about"))
	http.Handle("/metrics", promhttp.Handler())

	port := ":8080"
	log.Printf("Server starting on port %s", port)
	log.Printf("Endpoints available:")
	log.Printf("  GET /        - Hello World")
	log.Printf("  GET /health  - Health check")
	log.Printf("  GET /about   - About info")
	log.Printf("  GET /metrics - Prometheus metrics")
	
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
