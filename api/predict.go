package main

import (
	"encoding/csv"
	"math"
	"os"
	"strconv"
)

// boroughToFloat encodes the borough string as a numeric feature.
var boroughIndex = map[string]float64{
	"Bronx":         0,
	"Brooklyn":      1,
	"EWR":           2,
	"Manhattan":     3,
	"Queens":        4,
	"Staten Island": 5,
}

func predictTree(tree Tree, features [4]float64) float64 {
	if len(tree.Nodes) == 0 {
		return 0
	}
	idx := 0
	for {
		n := tree.Nodes[idx]
		if n.IsLeaf {
			return n.Value
		}
		if features[n.Feature] <= n.Threshold {
			idx = n.Left
		} else {
			idx = n.Right
		}
	}
}

func predictForest(forest []Tree, features [4]float64) float64 {
	if len(forest) == 0 {
		return 0
	}
	sum := 0.0
	for _, t := range forest {
		sum += predictTree(t, features)
	}
	return sum / float64(len(forest))
}

// calculateMetrics computes MAE, RMSE, and R² on a held-out test set.
// TODO: replace the nil call in main.go with your real test set from E1.
// The test set should be ~20% of clean data, never used in training.
func calculateMetrics(forest []Tree, testSet []TestSample) (mae, rmse, r2 float64) {
	if len(testSet) == 0 || len(forest) == 0 {
		return 0, 0, 0
	}

	n := float64(len(testSet))
	sumTarget := 0.0
	for _, s := range testSet {
		sumTarget += s.Target
	}
	meanTarget := sumTarget / n

	var sumAE, sumSE, ssTot float64
	for _, s := range testSet {
		pred := predictForest(forest, s.Features)
		diff := pred - s.Target
		sumAE += math.Abs(diff)
		sumSE += diff * diff
		d := s.Target - meanTarget
		ssTot += d * d
	}

	mae = sumAE / n
	rmse = math.Sqrt(sumSE / n)
	if ssTot > 0 {
		r2 = 1 - (sumSE / ssTot)
	}
	return
}

func loadTestSet(path string) ([]TestSample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Read() // skip header
	var samples []TestSample
	for {
		rec, err := r.Read()
		if err != nil {
			break
		}
		dist, _ := strconv.ParseFloat(rec[0], 64)
		hour, _ := strconv.ParseFloat(rec[1], 64)
		dow, _ := strconv.ParseFloat(rec[2], 64)
		bor := boroughIndex[rec[3]]
		dur, _ := strconv.ParseFloat(rec[4], 64)
		samples = append(samples, TestSample{
			Features: [4]float64{dist, hour, dow, bor},
			Target:   dur,
		})
	}
	return samples, nil
}
