#!/usr/bin/env perl

use strict;
use warnings;
use IO::Socket::INET;
use Time::HiRes qw(time);
use URI::Escape;
use POSIX qw(strftime);

package MetricsCollector {
    sub new {
        my $class = shift;
        my $self = {
            requests_total => {},
            request_durations => {}
        };
        return bless $self, $class;
    }
    
    sub record_request {
        my ($self, $method, $endpoint, $status_code, $duration) = @_;
        
        # Record request count
        my $key = "${method}_${endpoint}_${status_code}";
        $self->{requests_total}->{$key} = ($self->{requests_total}->{$key} || 0) + 1;
        
        # Record request duration
        my $duration_key = "${method}_${endpoint}";
        push @{$self->{request_durations}->{$duration_key}}, $duration;
    }
    
    sub get_metrics {
        my ($self) = @_;
        my $output = "";
        
        # HTTP requests total
        $output .= "# HELP http_requests_total Total number of HTTP requests\n";
        $output .= "# TYPE http_requests_total counter\n";
        
        for my $key (keys %{$self->{requests_total}}) {
            my ($method, $endpoint, $status_code) = split('_', $key, 3);
            my $count = $self->{requests_total}->{$key};
            $output .= "http_requests_total{method=\"$method\",endpoint=\"$endpoint\",status_code=\"$status_code\"} $count\n";
        }
        
        # HTTP request duration
        $output .= "# HELP http_request_duration_seconds Duration of HTTP requests in seconds\n";
        $output .= "# TYPE http_request_duration_seconds histogram\n";
        
        for my $key (keys %{$self->{request_durations}}) {
            my ($method, $endpoint) = split('_', $key, 2);
            my @durations = @{$self->{request_durations}->{$key}};
            my $count = scalar @durations;
            my $sum = 0;
            $sum += $_ for @durations;
            
            $output .= "http_request_duration_seconds_count{method=\"$method\",endpoint=\"$endpoint\"} $count\n";
            $output .= "http_request_duration_seconds_sum{method=\"$method\",endpoint=\"$endpoint\"} $sum\n";
        }
        
        return $output;
    }
}

package WebServer {
    sub new {
        my ($class, $port) = @_;
        $port ||= 9003;
        
        my $self = {
            port => $port,
            metrics => MetricsCollector->new(),
            socket => undef
        };
        
        return bless $self, $class;
    }
    
    sub start {
        my ($self) = @_;
        
        $self->{socket} = IO::Socket::INET->new(
            LocalPort => $self->{port},
            Type => SOCK_STREAM,
            Reuse => 1,
            Listen => 10
        ) or die "Could not create socket: $!";
        
        print "Server starting on port $self->{port}...\n";
        print "Metrics endpoint available at /metrics\n";
        
        while (my $client = $self->{socket}->accept()) {
            $self->handle_request($client);
            $client->close();
        }
    }
    
    sub handle_request {
        my ($self, $client) = @_;
        my $start_time = time();
        
        # Read request
        my $request = "";
        while (my $line = <$client>) {
            $request .= $line;
            last if $line =~ /^\r?\n$/;
        }
        
        return unless $request;
        
        # Parse request line
        my ($method, $path, $protocol) = split(' ', (split('\n', $request))[0]);
        $path ||= '/';
        $method ||= 'GET';
        
        # Remove query parameters for endpoint classification
        $path =~ s/\?.*$//;
        
        my $status_code = 200;
        my $response_body = "";
        my $content_type = "text/plain";
        
        # Route requests
        if ($path eq '/') {
            ($response_body, $content_type) = $self->handle_home();
        } elsif ($path eq '/post') {
            ($status_code, $response_body, $content_type) = $self->handle_post($method);
        } elsif ($path eq '/metrics') {
            ($response_body, $content_type) = $self->handle_metrics();
        } else {
            ($status_code, $response_body, $content_type) = $self->handle_404();
        }
        
        # Send response
        my $status_text = $self->get_status_text($status_code);
        my $response = "HTTP/1.1 $status_code $status_text\r\n";
        $response .= "Content-Type: $content_type\r\n";
        $response .= "Content-Length: " . length($response_body) . "\r\n";
        $response .= "Connection: close\r\n";
        $response .= "\r\n";
        $response .= $response_body;
        
        print $client $response;
        
        # Record metrics
        my $duration = time() - $start_time;
        $self->{metrics}->record_request($method, $path, $status_code, $duration);
    }
    
    sub handle_home {
        return ("Hello, World!", "text/plain");
    }
    
    sub handle_post {
        my ($self, $method) = @_;
        
        if ($method ne 'POST') {
            return (405, "Method not allowed", "text/plain");
        }
        
        return (200, "POST request received!", "text/plain");
    }
    
    sub handle_metrics {
        my ($self) = @_;
        my $metrics = $self->{metrics}->get_metrics();
        return ($metrics, "text/plain; version=0.0.4; charset=utf-8");
    }
    
    sub handle_404 {
        return (404, "Not Found", "text/plain");
    }
    
    sub get_status_text {
        my ($self, $code) = @_;
        
        my %status_texts = (
            200 => 'OK',
            404 => 'Not Found',
            405 => 'Method Not Allowed',
            500 => 'Internal Server Error'
        );
        
        return $status_texts{$code} || 'Unknown';
    }
    
    sub stop {
        my ($self) = @_;
        $self->{socket}->close() if $self->{socket};
    }
}

# Signal handlers for graceful shutdown
my $server;

$SIG{INT} = sub {
    print "\nShutting down server...\n";
    $server->stop() if $server;
    exit(0);
};

$SIG{TERM} = sub {
    print "\nShutting down server...\n";
    $server->stop() if $server;
    exit(0);
};

# Start the server
$server = WebServer->new(9003);
$server->start();
