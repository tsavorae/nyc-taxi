package main

import "math"

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
