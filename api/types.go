package main

import "time"

// Must match ml-node/types.go exactly.

type Task struct {
	TaskID      int `json:"task_id"`
	NumTrees    int `json:"num_trees"`
	MaxDepth    int `json:"max_depth"`
	MaxFeatures int `json:"max_features"`
}

type Node struct {
	IsLeaf    bool    `json:"is_leaf"`
	Feature   int     `json:"feature"`
	Threshold float64 `json:"threshold"`
	Value     float64 `json:"value"`
	Left      int     `json:"left"`
	Right     int     `json:"right"`
}

type Tree struct {
	Nodes []Node `json:"nodes"`
}

type Result struct {
	TaskID int    `json:"task_id"`
	NodeID string `json:"node_id"`
	Trees  []Tree `json:"trees"`
	DurMS  int64  `json:"dur_ms"`
	Error  string `json:"error,omitempty"`
}

// ─── API-specific types ───────────────────────────────────────────────────

type TrainRequest struct {
	TotalTrees  int `json:"total_trees"`
	MaxDepth    int `json:"max_depth"`
	MaxFeatures int `json:"max_features"`
}

// PredictRequest uses the 4 features selected by feature importance analysis.
type PredictRequest struct {
	TripDistance    float64 `json:"trip_distance"`
	PickupHour      float64 `json:"pickup_hour"`
	PickupDayOfWeek float64 `json:"pickup_day_of_week"`
	PUBorough       float64 `json:"pu_borough"` // encoded as int: 0=Bronx,1=Brooklyn,2=EWR,3=Manhattan,4=Queens,5=Staten Island
}

type NodeInfo struct {
	NodeID string `json:"node_id"`
	Trees  int    `json:"trees"`
	DurMS  int64  `json:"dur_ms"`
}

type TrainingMetrics struct {
	Trained    bool       `json:"trained"`
	TotalTrees int        `json:"total_trees"`
	DurTotalMS int64      `json:"dur_total_ms"`
	MAE        float64    `json:"mae"`
	RMSE       float64    `json:"rmse"`
	R2         float64    `json:"r2"`
	Timestamp  time.Time  `json:"timestamp"`
	PerNode    []NodeInfo `json:"per_node"`
}

// TestSample is used to evaluate the model after training.
// Features: [trip_distance, pickup_hour, pickup_day_of_week, pu_borough]
type TestSample struct {
	Features [4]float64
	Target   float64
}
