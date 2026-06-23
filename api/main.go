package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	log.Println("api escuchando en :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
