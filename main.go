package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
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
		[]string{"method", "endpoint", "status_code"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "Duration of HTTP requests in seconds",
		},
		[]string{"method", "endpoint"},
	)
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Initializing Prometheus metrics...")
	
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	
	log.Println("Prometheus metrics registered successfully")
}

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Incoming request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		
		// Create a response writer wrapper to capture status code
		ww := &responseWriter{ResponseWriter: w, statusCode: 200}
		
		next.ServeHTTP(ww, r)
		
		duration := time.Since(start).Seconds()
		log.Printf("Request completed: %s %s - Status: %d, Duration: %.3fs", 
			r.Method, r.URL.Path, ww.statusCode, duration)
		
		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, fmt.Sprintf("%d", ww.statusCode)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Serving hello endpoint")
	fmt.Fprintf(w, "Hello, World!")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Health check requested")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("About page requested")
	fmt.Fprintf(w, "This is a simple Go web server with Prometheus metrics")
}

func main() {
	log.Println("Starting application initialization...")
	
	mux := http.NewServeMux()
	
	log.Println("Registering HTTP handlers...")
	mux.HandleFunc("/", helloHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/about", aboutHandler)
	mux.Handle("/metrics", promhttp.Handler())
	log.Println("All handlers registered successfully")
	
	// Apply metrics middleware
	handler := metricsMiddleware(mux)
	log.Println("Metrics middleware applied")
	
	fmt.Println("Server starting on :8080")
	fmt.Println("Endpoints:")
	fmt.Println("  GET /        - Hello World")
	fmt.Println("  GET /health  - Health check")
	fmt.Println("  GET /about   - About page")
	fmt.Println("  GET /metrics - Prometheus metrics")
	
	log.Println("Starting HTTP server on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
