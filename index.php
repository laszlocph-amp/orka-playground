<?php

class MetricsCollector {
    private $requestsTotal = [];
    private $requestDurations = [];
    
    public function recordRequest($method, $endpoint, $statusCode, $duration) {
        $key = "{$method}_{$endpoint}_{$statusCode}";
        if (!isset($this->requestsTotal[$key])) {
            $this->requestsTotal[$key] = 0;
        }
        $this->requestsTotal[$key]++;
        
        $durationKey = "{$method}_{$endpoint}";
        if (!isset($this->requestDurations[$durationKey])) {
            $this->requestDurations[$durationKey] = [];
        }
        $this->requestDurations[$durationKey][] = $duration;
    }
    
    public function getMetrics() {
        $output = "";
        
        // HTTP requests total
        $output .= "# HELP http_requests_total Total number of HTTP requests\n";
        $output .= "# TYPE http_requests_total counter\n";
        foreach ($this->requestsTotal as $key => $count) {
            list($method, $endpoint, $statusCode) = explode('_', $key, 3);
            $output .= "http_requests_total{method=\"{$method}\",endpoint=\"{$endpoint}\",status_code=\"{$statusCode}\"} {$count}\n";
        }
        
        // HTTP request duration
        $output .= "# HELP http_request_duration_seconds Duration of HTTP requests in seconds\n";
        $output .= "# TYPE http_request_duration_seconds histogram\n";
        foreach ($this->requestDurations as $key => $durations) {
            list($method, $endpoint) = explode('_', $key, 2);
            $count = count($durations);
            $sum = array_sum($durations);
            $output .= "http_request_duration_seconds_count{method=\"{$method}\",endpoint=\"{$endpoint}\"} {$count}\n";
            $output .= "http_request_duration_seconds_sum{method=\"{$method}\",endpoint=\"{$endpoint}\"} {$sum}\n";
        }
        
        return $output;
    }
}

class WebServer {
    private $metrics;
    
    public function __construct() {
        $this->metrics = new MetricsCollector();
    }
    
    public function handleRequest() {
        $startTime = microtime(true);
        $method = $_SERVER['REQUEST_METHOD'];
        $path = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
        
        $statusCode = 200;
        
        switch ($path) {
            case '/':
                $this->handleHome();
                break;
            case '/post':
                $statusCode = $this->handlePost();
                break;
            case '/metrics':
                $this->handleMetrics();
                break;
            default:
                $statusCode = $this->handle404();
                break;
        }
        
        $duration = microtime(true) - $startTime;
        $this->metrics->recordRequest($method, $path, $statusCode, $duration);
    }
    
    private function handleHome() {
        header('Content-Type: text/plain');
        echo "Hello, World!";
    }
    
    private function handlePost() {
        if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
            http_response_code(405);
            header('Content-Type: text/plain');
            echo "Method not allowed";
            return 405;
        }
        
        header('Content-Type: text/plain');
        echo "POST request received!";
        return 200;
    }
    
    private function handleMetrics() {
        header('Content-Type: text/plain; version=0.0.4; charset=utf-8');
        echo $this->metrics->getMetrics();
    }
    
    private function handle404() {
        http_response_code(404);
        header('Content-Type: text/plain');
        echo "Not Found";
        return 404;
    }
}

// Initialize and handle the request
$server = new WebServer();
$server->handleRequest();

?>
