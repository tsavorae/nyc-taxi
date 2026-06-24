package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	mu      sync.RWMutex
	forest  []Tree
	metrics TrainingMetrics
	nodes   []string
)

func handleTrain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TrainRequest
	json.NewDecoder(r.Body).Decode(&req)

	// Defaults
	if req.TotalTrees == 0 {
		req.TotalTrees = 100
	}
	if req.MaxDepth == 0 {
		req.MaxDepth = 10
	}
	if req.MaxFeatures == 0 {
		req.MaxFeatures = 2 // sqrt(4 features) ≈ 2
	}

	log.Printf("POST /train — %d trees across %d nodes\n", req.TotalTrees, len(nodes))

	f, infos, dur, err := distributeTraining(nodes, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: replace nil with your real test set loaded from E1
	// Example: mae, rmse, r2 := calculateMetrics(f, testSet)
	mae, rmse, r2 := calculateMetrics(f, nil)

	mu.Lock()
	forest = f
	metrics = TrainingMetrics{
		Trained:    true,
		TotalTrees: len(f),
		DurTotalMS: dur,
		MAE:        mae,
		RMSE:       rmse,
		R2:         r2,
		Timestamp:  time.Now(),
		PerNode:    infos,
	}
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func handlePredict(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PredictRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	mu.RLock()
	f := forest
	mu.RUnlock()

	if len(f) == 0 {
		http.Error(w, "model not trained — call POST /train first", http.StatusConflict)
		return
	}

	features := [4]float64{
		req.TripDistance,
		req.PickupHour,
		req.PickupDayOfWeek,
		req.PUBorough,
	}

	pred := predictForest(f, features)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"trip_duration_min": math.Round(pred*100) / 100,
		"trees_used":        len(f),
	})
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mu.RLock()
	m := metrics
	mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}

func main() {
	nodesEnv := os.Getenv("NODES")
	if nodesEnv == "" {
		nodesEnv = "172.20.0.2:9051,172.20.0.3:9051,172.20.0.4:9051,172.20.0.5:9051"
	}
	nodes = strings.Split(nodesEnv, ",")
	log.Println("registered nodes:", nodes)

	http.HandleFunc("/train", handleTrain)
	http.HandleFunc("/predict", handlePredict)
	http.HandleFunc("/metrics", handleMetrics)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	log.Println("api listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
