package main

// Task is sent from the coordinator (api/) to each worker node.
type Task struct {
	TaskID      int `json:"task_id"`
	NumTrees    int `json:"num_trees"`
	MaxDepth    int `json:"max_depth"`
	MaxFeatures int `json:"max_features"`
}

// Node is a single node in a serialized decision tree.
type Node struct {
	IsLeaf    bool    `json:"is_leaf"`
	Feature   int     `json:"feature"`
	Threshold float64 `json:"threshold"`
	Value     float64 `json:"value"` // mean target if leaf
	Left      int     `json:"left"`  // index of left child  (-1 if leaf)
	Right     int     `json:"right"` // index of right child (-1 if leaf)
}

// Tree holds a flat slice of nodes for JSON serialization.
type Tree struct {
	Nodes []Node `json:"nodes"`
}

// Result is sent back from each worker to the coordinator.
type Result struct {
	TaskID int    `json:"task_id"`
	NodeID string `json:"node_id"`
	Trees  []Tree `json:"trees"`
	DurMS  int64  `json:"dur_ms"`
	Error  string `json:"error,omitempty"`
}

// Sample is one cleaned row of training data.
// Features: [trip_distance, pickup_hour, pickup_day_of_week, pu_borough]
type Sample struct {
	Features [4]float64
	Target   float64 // trip_duration in minutes
}
