# Go Hello World Server with Prometheus Metrics

A simple Go web server with multiple endpoints and Prometheus metrics integration.

## Features

- Hello World endpoint
- Health check endpoint
- About information endpoint
- Prometheus metrics collection
- HTTP request metrics (total requests, duration)

## Endpoints

- `GET /` - Returns "Hello, World! 🌍"
- `GET /health` - Health check endpoint
- `GET /about` - Server information
- `GET /metrics` - Prometheus metrics

## Running

```bash
go mod tidy
go run main.go
```

Server will start on port 8080.

## Metrics

The server collects the following Prometheus metrics:

- `http_requests_total` - Counter of total HTTP requests by endpoint, method, and status
- `http_request_duration_seconds` - Histogram of HTTP request durations

Access metrics at: http://localhost:8080/metrics
