package main

import (
	"log"
	"net"
	"os"
)

func main() {
	nodeID := os.Getenv("NODE_ID")
	ln, err := net.Listen("tcp", ":9051")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("[node-%s] escuchando en :9051\n", nodeID)
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		log.Printf("[node-%s] conexión recibida de %s\n", nodeID, conn.RemoteAddr())
		conn.Close()
	}
}
