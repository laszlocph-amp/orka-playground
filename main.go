package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

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

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}
		
		response := fmt.Sprintf("Echo: %s", string(message))
		err = conn.WriteMessage(messageType, []byte(response))
		if err != nil {
			log.Printf("Error writing message: %v", err)
			break
		}
	}
}

func main() {
	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/post", postHandler)
	http.HandleFunc("/ws", websocketHandler)
	fmt.Println("Server starting on port 9001...")
	fmt.Println("WebSocket endpoint available at /ws")
	log.Fatal(http.ListenAndServe(":9001", nil))
}
