package main

import (
	"fmt"
	"log"
	"net/http"
)

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

func main() {
	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/post", postHandler)
	fmt.Println("Server starting on port 9000...")
	log.Fatal(http.ListenAndServe(":9000", nil))
}
