#!/usr/bin/env perl

# HTTP Webserver mit Prometheus-Style Metriken
# Implementiert in Perl mit einfachem Socket-basiertem Server

use strict;
use warnings;
use IO::Socket::INET;
use Time::HiRes qw(time);
use URI::Escape;
use POSIX qw(strftime);

# Metriken-Sammler Klasse
# Verwaltet HTTP Request Zähler und Antwortzeiten
package MetricsCollector {
    sub new {
        my $class = shift;
        my $self = {
            requests_total => {},      # Gesamt-Anzahl der HTTP Requests
            request_durations => {}    # Antwortzeiten für jeden Request
        };
        return bless $self, $class;
    }
    
    # Zeichnet einen HTTP Request mit Metriken auf
    # Parameter: Methode, Endpoint, Status-Code, Dauer
    sub record_request {
        my ($self, $method, $endpoint, $status_code, $duration) = @_;
        
        # Request-Zähler erhöhen
        my $key = "${method}_${endpoint}_${status_code}";
        $self->{requests_total}->{$key} = ($self->{requests_total}->{$key} || 0) + 1;
        
        # Request-Dauer speichern
        my $duration_key = "${method}_${endpoint}";
        push @{$self->{request_durations}->{$duration_key}}, $duration;
    }
    
    # Generiert Prometheus-Format Metriken
    # Gibt einen String mit allen gesammelten Metriken zurück
    sub get_metrics {
        my ($self) = @_;
        my $output = "";
        
        # HTTP Request Gesamtzähler ausgeben
        $output .= "# HELP http_requests_total Gesamtanzahl der HTTP Requests\n";
        $output .= "# TYPE http_requests_total counter\n";
        
        for my $key (keys %{$self->{requests_total}}) {
            my ($method, $endpoint, $status_code) = split('_', $key, 3);
            my $count = $self->{requests_total}->{$key};
            $output .= "http_requests_total{method=\"$method\",endpoint=\"$endpoint\",status_code=\"$status_code\"} $count\n";
        }
        
        # HTTP Request Dauer Histogramm ausgeben
        $output .= "# HELP http_request_duration_seconds Dauer der HTTP Requests in Sekunden\n";
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

# Webserver Hauptklasse
# Behandelt HTTP Requests und verwaltet Socket-Verbindungen
package WebServer {
    sub new {
        my ($class, $port) = @_;
        $port ||= 9003;  # Standard-Port falls nicht angegeben
        
        my $self = {
            port => $port,
            metrics => MetricsCollector->new(),  # Metriken-Sammler initialisieren
            socket => undef
        };
        
        return bless $self, $class;
    }
    
    # Startet den HTTP Server
    # Erstellt Socket und wartet auf eingehende Verbindungen
    sub start {
        my ($self) = @_;
        
        # TCP Socket erstellen und auf Port binden
        $self->{socket} = IO::Socket::INET->new(
            LocalPort => $self->{port},
            Type => SOCK_STREAM,
            Reuse => 1,     # Port-Wiederverwendung erlauben
            Listen => 10    # Warteschlange für eingehende Verbindungen
        ) or die "Socket konnte nicht erstellt werden: $!";
        
        print "Server startet auf Port $self->{port}...\n";
        print "Metriken-Endpoint verfügbar unter /metrics\n";
        
        # Hauptschleife: Warte auf Client-Verbindungen
        while (my $client = $self->{socket}->accept()) {
            $self->handle_request($client);
            $client->close();
        }
    }
    
    # Behandelt einen einzelnen HTTP Request
    # Parst Request, routet zu entsprechendem Handler, sendet Antwort
    sub handle_request {
        my ($self, $client) = @_;
        my $start_time = time();  # Startzeit für Latenz-Messung
        
        # HTTP Request vom Client lesen
        my $request = "";
        while (my $line = <$client>) {
            $request .= $line;
            last if $line =~ /^\r?\n$/;  # Leerzeile = Ende der HTTP Headers
        }
        
        return unless $request;  # Abbruch falls kein Request empfangen
        
        # Request-Zeile parsen (erste Zeile des HTTP Requests)
        my ($method, $path, $protocol) = split(' ', (split('\n', $request))[0]);
        $path ||= '/';       # Standard-Pfad falls nicht angegeben
        $method ||= 'GET';   # Standard-Methode falls nicht angegeben
        
        # Query-Parameter für Endpoint-Klassifizierung entfernen
        $path =~ s/\?.*$//;
        
        my $status_code = 200;
        my $response_body = "";
        my $content_type = "text/plain";
        
        # Request zu entsprechendem Handler weiterleiten
        if ($path eq '/') {
            ($response_body, $content_type) = $self->handle_home();
        } elsif ($path eq '/post') {
            ($status_code, $response_body, $content_type) = $self->handle_post($method);
        } elsif ($path eq '/metrics') {
            ($response_body, $content_type) = $self->handle_metrics();
        } else {
            ($status_code, $response_body, $content_type) = $self->handle_404();
        }
        
        # HTTP Response zusammenstellen und senden
        my $status_text = $self->get_status_text($status_code);
        my $response = "HTTP/1.1 $status_code $status_text\r\n";
        $response .= "Content-Type: $content_type\r\n";
        $response .= "Content-Length: " . length($response_body) . "\r\n";
        $response .= "Connection: close\r\n";
        $response .= "\r\n";
        $response .= $response_body;
        
        print $client $response;
        
        # Metriken für diesen Request aufzeichnen
        my $duration = time() - $start_time;
        $self->{metrics}->record_request($method, $path, $status_code, $duration);
    }
    
    # Handler für Root-Endpoint ("/")
    # Gibt einfache "Hello, World!" Nachricht zurück
    sub handle_home {
        return ("Hello, World!", "text/plain");
    }
    
    # Handler für POST-Endpoint ("/post")
    # Überprüft HTTP-Methode und gibt entsprechende Antwort
    sub handle_post {
        my ($self, $method) = @_;
        
        if ($method ne 'POST') {
            return (405, "Methode nicht erlaubt", "text/plain");
        }
        
        return (200, "POST Request empfangen!", "text/plain");
    }
    
    # Handler für Metriken-Endpoint ("/metrics")
    # Gibt Prometheus-Format Metriken zurück
    sub handle_metrics {
        my ($self) = @_;
        my $metrics = $self->{metrics}->get_metrics();
        return ($metrics, "text/plain; version=0.0.4; charset=utf-8");
    }
    
    # Handler für nicht gefundene Pfade (404)
    sub handle_404 {
        return (404, "Nicht gefunden", "text/plain");
    }
    
    # Konvertiert HTTP Status-Code zu Text
    # Gibt entsprechenden Status-Text für HTTP Response zurück
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
    
    # Stoppt den Server und schließt Socket
    sub stop {
        my ($self) = @_;
        $self->{socket}->close() if $self->{socket};
    }
}

# Signal-Handler für ordnungsgemäßes Herunterfahren
# Reagiert auf SIGINT (Ctrl+C) und SIGTERM
my $server;

$SIG{INT} = sub {
    print "\nServer wird heruntergefahren...\n";
    $server->stop() if $server;
    exit(0);
};

$SIG{TERM} = sub {
    print "\nServer wird heruntergefahren...\n";
    $server->stop() if $server;
    exit(0);
};

# Server-Instanz erstellen und starten
# Läuft auf Port 9003 mit Hello World und Metriken-Endpoints
$server = WebServer->new(9003);
$server->start();
