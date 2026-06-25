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

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	mu      sync.RWMutex
	forest  []Tree
	metrics TrainingMetrics
	nodes   []string
)
var testSet []TestSample

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

	// Broadcast training start to WebSocket clients
	hub.Broadcast(WSEvent{Type: "train_start", Data: TrainStartEvent{
		TotalTrees: req.TotalTrees,
		NumNodes:   len(nodes),
	}})

	f, infos, dur, err := distributeTraining(nodes, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Example: mae, rmse, r2 := calculateMetrics(f, testSet)
	mae, rmse, r2 := calculateMetrics(f, testSet)

	// Broadcast per-node results
	for _, info := range infos {
		hub.Broadcast(WSEvent{Type: "node_done", Data: NodeDoneEvent{
			NodeID: info.NodeID,
			Trees:  info.Trees,
			DurMS:  info.DurMS,
		}})
	}

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

	// Store training record in MongoDB
	claims := getClaims(r)
	if claims != nil {
		uid, _ := primitive.ObjectIDFromHex(claims.UserID)
		rec := &TrainRecord{
			UserID:     uid,
			TotalTrees: len(f),
			MAE:        mae,
			RMSE:       rmse,
			R2:         r2,
			DurTotalMS: dur,
			PerNode:    infos,
		}
		if err := insertTrainRecord(rec); err != nil {
			log.Println("failed to store train record:", err)
		}
	}

	// Broadcast training done
	hub.Broadcast(WSEvent{Type: "train_done", Data: TrainDoneEvent{
		TotalTrees: len(f),
		DurTotalMS: dur,
		MAE:        mae,
		RMSE:       rmse,
		R2:         r2,
	}})

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

func handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims := getClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"no claims"}`, http.StatusUnauthorized)
		return
	}

	uid, _ := primitive.ObjectIDFromHex(claims.UserID)
	records, err := listTrainHistory(uid, 50)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

func main() {
	// Infrastructure
	connectMongo()
	connectRedis()
	initAuth()

	var err error
	testSet, err = loadTestSet("/data/processed/test_data.csv")
	if err != nil {
		log.Println("warning: test set not loaded:", err)
	}

	nodesEnv := os.Getenv("NODES")
	if nodesEnv == "" {
		nodesEnv = "172.20.0.2:9051,172.20.0.3:9051,172.20.0.4:9051,172.20.0.5:9051"
	}
	nodes = strings.Split(nodesEnv, ",")
	log.Println("registered nodes:", nodes)

	// Public endpoints
	http.HandleFunc("/register", handleRegister)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	// WebSocket (public — auth via query param if needed)
	http.HandleFunc("/ws", handleWS)

	// Protected endpoints
	http.HandleFunc("/train", authMiddleware(handleTrain))
	http.HandleFunc("/predict", authMiddleware(handlePredict))
	http.HandleFunc("/metrics", authMiddleware(handleMetrics))
	http.HandleFunc("/logout", authMiddleware(handleLogout))
	http.HandleFunc("/history", authMiddleware(handleHistory))

	log.Println("api listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
