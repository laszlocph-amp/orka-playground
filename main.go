package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan []byte)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World!")
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fmt.Fprintf(w, "POST request received!")
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	clients[conn] = true
	log.Printf("Client connected. Total clients: %d", len(clients))

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			delete(clients, conn)
			break
		}
		
		timestamp := time.Now().Format("15:04:05")
		formattedMessage := fmt.Sprintf("[%s] %s", timestamp, string(message))
		broadcast <- []byte(formattedMessage)
	}
}

func handleMessages() {
	for {
		message := <-broadcast
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("Error writing message: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func main() {
	go handleMessages()
	
	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/post", postHandler)
	http.HandleFunc("/chat", chatHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("."))))
	fmt.Println("Server starting on port 9002...")
	fmt.Println("Chat WebSocket endpoint available at /chat")
	fmt.Println("Test client available at http://localhost:9002/static/chat.html")
	log.Fatal(http.ListenAndServe(":9002", nil))
