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
	
	log.Println("Main server starting on :8080")
	log.Println("Metrics server available on :9090/metrics")
	log.Fatal(http.ListenAndServe(":8080", mainServeMux))
}
