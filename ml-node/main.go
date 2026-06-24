package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net"
	"os"
	"time"
)

var data []Sample // loaded once at startup, read-only after that

func handle(conn net.Conn) {
	defer conn.Close()

	var task Task
	if err := json.NewDecoder(conn).Decode(&task); err != nil {
		log.Println("error decoding task:", err)
		return
	}

	nodeID := os.Getenv("NODE_ID")
	log.Printf("[node-%s] task %d received — training %d trees\n", nodeID, task.TaskID, task.NumTrees)

	start := time.Now()
	trees := make([]Tree, task.NumTrees)
	for i := range trees {
		trees[i] = trainTree(data, task.MaxDepth, task.MaxFeatures)
	}
	dur := time.Since(start).Milliseconds()

	log.Printf("[node-%s] task %d completed in %dms\n", nodeID, task.TaskID, dur)

	res := Result{
		TaskID: task.TaskID,
		NodeID: "node-" + nodeID,
		Trees:  trees,
		DurMS:  dur,
	}
	if err := json.NewEncoder(conn).Encode(res); err != nil {
		log.Println("error sending result:", err)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	path := os.Getenv("DATA_PARQUET")
	if path == "" {
		path = "/data/processed/yellow_tripdata_2026_clean.parquet"
	}

	var err error
	data, err = loadData(path)
	if err != nil {
		log.Fatalf("error loading data: %v", err)
	}

	ln, err := net.Listen("tcp", ":9051")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	log.Println("ml-node listening on :9051")

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("error accepting connection:", err)
			continue
		}
		go handle(conn)
	}
}
