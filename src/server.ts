import * as http from 'http';
import * as url from 'url';
import { register, Counter, Histogram } from 'prom-client';

interface MetricsData {
    httpRequestsTotal: Counter<string>;
    httpRequestDuration: Histogram<string>;
}

class WebServer {
    private server: http.Server;
    private metrics: MetricsData;
    
    constructor(private port: number = 9003) {
        this.initializeMetrics();
        this.server = http.createServer(this.handleRequest.bind(this));
    }
    
    private initializeMetrics(): void {
        this.metrics = {
            httpRequestsTotal: new Counter({
                name: 'http_requests_total',
                help: 'Total number of HTTP requests',
                labelNames: ['method', 'endpoint', 'status_code']
            }),
            
            httpRequestDuration: new Histogram({
                name: 'http_request_duration_seconds',
                help: 'Duration of HTTP requests in seconds',
                labelNames: ['method', 'endpoint']
            })
        };
        
        register.registerMetric(this.metrics.httpRequestsTotal);
        register.registerMetric(this.metrics.httpRequestDuration);
    }
    
    private async handleRequest(req: http.IncomingMessage, res: http.ServerResponse): Promise<void> {
        const startTime = Date.now();
        const method = req.method || 'GET';
        const parsedUrl = url.parse(req.url || '/', true);
        const pathname = parsedUrl.pathname || '/';
        
        let statusCode = 200;
        
        try {
            switch (pathname) {
                case '/':
                    await this.handleHome(req, res);
                    break;
                case '/post':
                    statusCode = await this.handlePost(req, res);
                    break;
                case '/metrics':
                    await this.handleMetrics(req, res);
                    break;
                default:
                    statusCode = await this.handle404(req, res);
                    break;
            }
        } catch (error) {
            console.error('Error handling request:', error);
            statusCode = 500;
            res.writeHead(500, { 'Content-Type': 'text/plain' });
            res.end('Internal Server Error');
        }
        
        const duration = (Date.now() - startTime) / 1000;
        this.recordMetrics(method, pathname, statusCode, duration);
    }
    
    private recordMetrics(method: string, endpoint: string, statusCode: number, duration: number): void {
        this.metrics.httpRequestsTotal
            .labels(method, endpoint, statusCode.toString())
            .inc();
            
        this.metrics.httpRequestDuration
            .labels(method, endpoint)
            .observe(duration);
    }
    
    private async handleHome(req: http.IncomingMessage, res: http.ServerResponse): Promise<void> {
        res.writeHead(200, { 'Content-Type': 'text/plain' });
        res.end('Hello, World!');
    }
    
    private async handlePost(req: http.IncomingMessage, res: http.ServerResponse): Promise<number> {
        if (req.method !== 'POST') {
            res.writeHead(405, { 'Content-Type': 'text/plain' });
            res.end('Method not allowed');
            return 405;
        }
        
        res.writeHead(200, { 'Content-Type': 'text/plain' });
        res.end('POST request received!');
        return 200;
    }
    
    private async handleMetrics(req: http.IncomingMessage, res: http.ServerResponse): Promise<void> {
        try {
            const metrics = await register.metrics();
            res.writeHead(200, { 'Content-Type': register.contentType });
            res.end(metrics);
        } catch (error) {
            res.writeHead(500, { 'Content-Type': 'text/plain' });
            res.end('Error generating metrics');
        }
    }
    
    private async handle404(req: http.IncomingMessage, res: http.ServerResponse): Promise<number> {
        res.writeHead(404, { 'Content-Type': 'text/plain' });
        res.end('Not Found');
        return 404;
    }
    
    public start(): void {
        this.server.listen(this.port, () => {
            console.log(`Server starting on port ${this.port}...`);
            console.log(`Metrics endpoint available at /metrics`);
        });
    }
    
    public stop(): void {
        this.server.close();
    }
}

// Start the server
const server = new WebServer(9003);
server.start();

// Graceful shutdown
process.on('SIGINT', () => {
    console.log('\nShutting down server...');
    server.stop();
    process.exit(0);
});

process.on('SIGTERM', () => {
    console.log('\nShutting down server...');
    server.stop();
    process.exit(0);
});
